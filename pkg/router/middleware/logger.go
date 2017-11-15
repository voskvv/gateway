package middleware

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/cactus/go-statsd-client/statsd"
)


//TODO: Move Statter to another file
//Statter connection with Statsd
var Statter *statsd.Statter

type LoggerResponseWritter interface {
	http.ResponseWriter
	Status() int
	BytesWritten() int
}

type loggerWritter struct {
	http.ResponseWriter
	wroteHeader bool
	code        int
	bytes       int
	headers     []string
}

func NewLoggerResponseWritter(w http.ResponseWriter) LoggerResponseWritter {
	return &loggerWritter{ResponseWriter: w}
}

func (lw *loggerWritter) WriteHeader(code int) {
	if !lw.wroteHeader {
		lw.code = code
		lw.wroteHeader = true
		lw.ResponseWriter.WriteHeader(code)
	}
}

func (lw *loggerWritter) Write(buf []byte) (int, error) {
	lw.WriteHeader(http.StatusOK)
	n, err := lw.ResponseWriter.Write(buf)
	lw.bytes += n
	return n, err
}

func (lw *loggerWritter) Status() int {
	return lw.code
}

func (lw *loggerWritter) BytesWritten() int {
	return lw.bytes
}

//Logger write main logs
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := NewLoggerResponseWritter(w)

		next.ServeHTTP(lw, r)
		
		latency := time.Now().Sub(start)

		//Set status in Statsd
		if Statter != nil {
			statusCall := fmt.Sprintf("call.status.%v", lw.Status())
			methodCall := fmt.Sprintf("call.method.%v", r.Method)
			(*Statter).Inc("call.status.all", 1, 1.0)
			(*Statter).Inc(statusCall, 1, 1.0)
			(*Statter).Inc(methodCall, 1, 1.0)
		}

		//Write log after
		log.WithFields(log.Fields{
			"Method":       r.Method,
			"Path":         r.RequestURI,
			"Latency":      fmt.Sprintf("%v", latency),
			"Status":       lw.Status(),
			"RequestID":    w.Header().Get("X-Request-ID"),
			"ResponseSize": lw.BytesWritten(),
			"RequestSize":  r.ContentLength,
		}).Info("Request")
	})
}
