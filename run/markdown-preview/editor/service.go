package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// MarkdownRenderer defines an interface for rendering Markdown to HTML.
type MarkdownRenderer interface {
	Render([]byte) ([]byte, error)
}

// Service manages centralized resources of the service
type Service struct {
	Renderer MarkdownRenderer

	parsedTemplate  *template.Template
	markdownDefault string
}

// NewServiceFromEnv creates a new Service instance from environment variables.
func NewServiceFromEnv() (*Service, error) {
	url := os.Getenv("EDITOR_UPSTREAM_RENDER_URL")
	if url == "" {
		return nil, fmt.Errorf("no configuration for upstream render service: add EDITOR_UPSTREAM_RENDER_URL environment variable")
	}
	auth := os.Getenv("EDITOR_UPSTREAM_UNAUTHENTICATED") == ""
	if !auth {
		log.Println("editor: starting in unauthenticated upstream mode")
	}

	// The use case of this service is the UI driven by these files.
	// Loading them as part of the server startup process keeps failures easily
	// discoverable and minimizes latency for the first request.
	parsedTemplate, err := template.ParseFiles("templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("template.ParseFiles: %w", err)
	}

	out, err := ioutil.ReadFile("templates/markdown.md")
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile: %w", err)
	}
	markdownDefault := string(out)

	return &Service{
		Renderer: &RenderService{
			URL:           url,
			Authenticated: auth,
		},
		parsedTemplate:  parsedTemplate,
		markdownDefault: markdownDefault,
	}, nil
}

// RegisterHandlers registers all HTTP handler routes to a new ServeMux.
func (s *Service) RegisterHandlers() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.editorHandler)
	mux.HandleFunc("/render", s.renderHandler)

	return mux
}

func (s *Service) editorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := s.parsedTemplate.Execute(w, map[string]string{"Default": s.markdownDefault}); err != nil {
		log.Printf("template.Execute: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Service) renderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	out, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var d struct{ Data string }
	if err := json.Unmarshal(out, &d); err != nil {
		log.Printf("json.Unmarshal: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	rendered, err := s.Renderer.Render([]byte(d.Data))
	if err != nil {
		log.Printf("MarkdownRenderer.Render: %v", err)
		if strings.Contains(err.Error(), "metadata.Get") {
			log.Printf("If running locally try restarting with the environment variable 'EDITOR_UPSTREAM_UNAUTHENTICATED=1'")
		}

		msg := http.StatusText(http.StatusInternalServerError)
		if errors.Is(err, errNotOk) {
			msg = fmt.Sprintf("<h3>%s (%d)</h3>\n<p>The request to the upstream render service failed with the message:</p>\n<p>%s</p>", http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError, rendered)
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.Write(rendered)
}
