package serializer

import "encoding/json"

type RequestRawSign struct {
	Path   string
	Header string
	Body   string
}

func NewRequestSignString(path, header, body string) string {
	req := RequestRawSign{
		Path:   path,
		Header: header,
		Body:   body,
	}
	res, _ := json.Marshal(req)
	return string(res)
}
