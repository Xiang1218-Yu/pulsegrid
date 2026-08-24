package httpapi

import "net/http"

func JSONHeaders(writer http.ResponseWriter) {
	writer.Header().Set("content-type", "application/json; charset=utf-8")
}

func EventHeaders(writer http.ResponseWriter) {
	writer.Header().Set("content-type", "application/x-ndjson")
	writer.Header().Set("cache-control", "no-cache")
}

func DownloadHeaders(writer http.ResponseWriter, filename string) {
	writer.Header().Set("content-disposition", "attachment; filename="+filename)
}
