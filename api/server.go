package api

import (
	"net"
	"net/http"

	"github.com/glow-mdsol/dovin/store"
)

type Server struct {
	store    *store.Store
	listener net.Listener
	port     int
}

func New(s *store.Store) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &Server{
		store:    s,
		listener: ln,
		port:     ln.Addr().(*net.TCPAddr).Port,
	}
	return srv, nil
}

func (s *Server) Port() int { return s.port }

func (s *Server) Handler(uiFS http.FileSystem) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(uiFS))
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/tasks/", s.handleTask)
	mux.HandleFunc("/recurrences", s.handleRecurrences)
	mux.HandleFunc("/recurrences/", s.handleRecurrence)
	mux.HandleFunc("/notes", s.handleNotes)
	mux.HandleFunc("/notes/", s.handleNote)
	return mux
}

func (s *Server) Serve(h http.Handler) error {
	return http.Serve(s.listener, h)
}
