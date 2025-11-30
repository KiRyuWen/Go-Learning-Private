package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Middleware struct { //middleware handler
	mux http.Handler
}

type DBServerHandler struct { //handler
	db *sql.DB
}

func NewDBServerHandler(db *sql.DB) DBServerHandler {
	return DBServerHandler{db}
}

func (s *DBServerHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	searchUniName := request.URL.Query().Get("q")

	if searchUniName == "" {
		if _, err := writer.Write([]byte("Should include university name, instead of empty")); err != nil {
			fmt.Println("error when writing response for /search request")
			writer.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	results, err := SearchSchoolsDB(s.db, searchUniName)

	if err != nil {
		fmt.Println("error when query db name: ", err)
		writer.WriteHeader(http.StatusInternalServerError)
	}

	writer.Header().Set("Content-Type", "application/json")

	json.NewEncoder(writer).Encode(results)

}

func (m Middleware) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	//start timer
	ctx := context.WithValue(r.Context(), "user", "unknown")
	ctx = context.WithValue(ctx, "__requestStartTimer__", time.Now())
	req := r.WithContext(ctx)

	//exec handler
	m.mux.ServeHTTP(rw, req)
	//end timer
	start := req.Context().Value("__requestStartTimer__").(time.Time)
	fmt.Println("request duration: ", time.Since(start))
}

func startDBServer(db *sql.DB) {

	//register a DB mux
	dbServerHandler := NewDBServerHandler(db)

	dbServerMux := http.NewServeMux()
	dbServerMux.Handle("/search", &dbServerHandler)

	//setup server
	dbServer := &http.Server{
		Addr:         ":8080",
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		Handler:      Middleware{dbServerMux},
	}

	dbServer.ListenAndServe()
}
