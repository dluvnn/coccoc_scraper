package libs

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
)

func WaitCtrlC() {
	cwait := make(chan os.Signal, 1)
	signal.Notify(cwait, os.Interrupt)
	<-cwait
}

type CheckAPIKeyFn func(http.ResponseWriter, *http.Request) bool

func MakeCheckAPIKey(apiKey string) CheckAPIKeyFn {
	if len(apiKey) == 0 {
		return func(w http.ResponseWriter, r *http.Request) bool {
			return true
		}
	}

	k := apiKey
	return func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("api-key") != k {
			http.Error(w, "incorrect API key", http.StatusUnauthorized)
			return false
		}
		return true
	}
}

func ReadBody(r *http.Request) ([]byte, error) {
	data, err := io.ReadAll(r.Body)
	r.Body.Close()
	return data, err
}

// JSONParse parses the request body as JSON object
func JSONParse(r *http.Request, x interface{}) error {
	data, err := ReadBody(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, x)
}

// JSONReply writes reponse with the body is JSON of variable x
func JSONReply(w http.ResponseWriter, x interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(x)
}

func ServerError(w http.ResponseWriter, err error, status_code int) {
	http.Error(w, err.Error(), status_code)
}

func InternalServerError(w http.ResponseWriter, err error) {
	ServerError(w, err, http.StatusInternalServerError)
}

func BadRequest(w http.ResponseWriter, err error) {
	ServerError(w, err, http.StatusBadRequest)
}

func Unique[T comparable](v []T) []T {
	m := map[T]bool{}
	n := len(v)
	for i := 0; i < n; i++ {
		m[v[i]] = true
	}

	n = len(m)
	rs := make([]T, n)
	i := 0
	for k := range m {
		rs[i] = k
		i++
	}
	return rs
}

func Serve(port int) {
	log.Printf("start listen at :%d", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic(err)
	}
}

// FileExists checks a file exists or not.
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
