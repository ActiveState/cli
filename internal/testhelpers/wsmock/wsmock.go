package wsmock

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/gorilla/websocket"
	"github.com/posener/wstest"
)

type WsMock struct {
	upgrader      websocket.Upgrader
	responders    map[string]string
	responsePath  string
	responseQueue []string
	done          chan bool
}

func Init() *WsMock {
	mock := &WsMock{
		responders: map[string]string{},
	}
	mock.upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	return mock
}

func (s *WsMock) Dialer() *websocket.Dialer {
	return wstest.NewDialer(s)
}

func (s *WsMock) RegisterWithResponse(requestContains string, responseFile string) {
	s.responders[requestContains] = responseFile
}

func (s *WsMock) QueueResponse(responseFile string) {
	responseFile = s.getResponseFile(responseFile)
	response := string(fileutils.ReadFileUnsafe(responseFile))
	s.responseQueue = append(s.responseQueue, response)
}

func (s *WsMock) Close() {
	logging.Debug("Close called")
	s.done <- true
}

func (s *WsMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.done = make(chan bool)

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(fmt.Sprintf("Could not upgrade connection to websocket: %v", err))
	}

	for s.done != nil {
		logging.Debug("Loop")
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if strings.HasPrefix(err.Error(), "websocket: close") {
				logging.Debug("websocket close encountered")
				s.Close()
				return
			}
			multilog.Error("Reading Message failed: %v", err)
			return
		}

		msg := string(msgBytes[:])
		logging.Debug("Message received: %v", msg)

		for requestContains, responseFile := range s.responders {
			if strings.Contains(msg, requestContains) {
				responseFile = s.getResponseFile(responseFile)
				response := string(fileutils.ReadFileUnsafe(responseFile))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
					panic(fmt.Sprintf("Could not write response to websocket: %v", err))
				}
				break
			}
		}

		for _, response := range s.responseQueue {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
				panic(fmt.Sprintf("Could not write response to websocket: %v", err))
			}
		}
		s.responseQueue = []string{}
	}
}

func (s *WsMock) getResponseFile(responseFile string) string {
	return filepath.Join(s.getResponsePath(), responseFile) + ".json"
}

func (s *WsMock) getResponsePath() string {
	if s.responsePath == "" {
		_, currentFile, _, _ := runtime.Caller(0)
		file := currentFile
		ok := true
		iter := 2

		for file == currentFile && ok {
			_, file, _, ok = runtime.Caller(iter)
			iter = iter + 1
		}

		if file == "" || file == currentFile {
			panic("Could not get caller")
		}
		s.responsePath = filepath.Join(filepath.Dir(file), "testdata", "wsresponse")
	}

	return s.responsePath
}
