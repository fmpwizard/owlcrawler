package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//messageStore 's key is sessionId + cometId'
var messageStore = struct {
	sync.RWMutex
	LastIndex uint64
	m         map[sessionCometKey][]message
}{m: make(map[sessionCometKey][]message)}

//cometStore 's key is sessionId'
var cometStore = struct {
	sync.RWMutex
	m map[session]comet
}{m: make(map[session]comet)}

type message struct {
	index uint64
	Value jsCmd `json:"value"`
	Stamp time.Time
}

type comet struct {
	Value    string
	LastSeen time.Time
}

type TemplateInfo struct {
	CometId  string
	Index    uint64
	Messages []message
}

type Response struct {
	Value jsCmd `json:"value"`
	Error string
}

type Responses struct {
	Res       []Response `json:"resp"`
	LastIndex uint64     `json:"lastIndex"`
	Event     string     `json:"event"`
}

type sessionCometKey string

type session string

type jsCmd struct {
	Js string `json:"js"`
}

///////////////////

type Message struct {
	Id        string `json:"id"`
	Body      string `json:"body"`
	CreatedOn int64  `json:"createdOn"`
}

var rootDir string

func init() {
	currentDir, _ := os.Getwd()
	flag.StringVar(&rootDir, "root-dir", currentDir, "specifies the root dir where html and other files will be relative to")
}

func main() {
	flag.Parse()
	http.HandleFunc("/index", showMessages)
	http.HandleFunc("/api/messages/new", createChatMessage)
	http.HandleFunc("/api/messages/page", retrieveChatMessages)
	http.HandleFunc("/api/comet", handleComet)
	http.Handle("/bower_components/", http.StripPrefix("/bower_components/", http.FileServer(http.Dir("app/bower_components"))))
	http.Handle("/build/", http.StripPrefix("/build/", http.FileServer(http.Dir("build"))))
	go gc()
	log.Println("Listening ...")
	log.Fatal(http.ListenAndServe(":7070", nil))
}

func createChatMessage(rw http.ResponseWriter, req *http.Request) {
	guid, err := uuid.NewV4()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	data := Message{Id: guid.String()}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("Error reading Body, got %v", err)
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("4 error ", err)
	}

	ret := "console.log('" + data.Body + "');"
	currentComet := req.FormValue("cometid") //TODO make sure to pass this in
	fmt.Printf("got currentComet  %v\n", currentComet)
	cookie, _ := req.Cookie("gsessionid")
	messageStore.Lock()
	messageStore.LastIndex++
	messageStore.m[sessionCometKey(cookie.Value+currentComet)] = append(messageStore.m[sessionCometKey(cookie.Value+currentComet)], message{messageStore.LastIndex, jsCmd{ret}, time.Now()})
	messageStore.Unlock()

	jsonRet, err := json.Marshal(map[string]string{"id": guid.String()})
	if err != nil {
		fmt.Printf("Error marshalling %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Header().Add("Content-Type", "text/plain")
	} else {
		rw.WriteHeader(http.StatusCreated)
		rw.Write(jsonRet)

	}

}

func retrieveChatMessages(rw http.ResponseWriter, req *http.Request) {
	page, err := strconv.ParseInt(req.FormValue("page"), 10, 0)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		fmt.Printf("getting page %d\n", page)
	}

}

func showMessages(rw http.ResponseWriter, req *http.Request) {
	t := template.New("index.html")
	t.Funcs(template.FuncMap{"UnixToString": UnixToString})
	t, err := t.ParseFiles(path.Join(rootDir, "app/index.html"))
	if err != nil {
		fmt.Printf("Error parsing template files: %v", err)
	}
	cookie, err := req.Cookie("gsessionid")
	if err == http.ErrNoCookie {
		rand.Seed(time.Now().UnixNano())
		sess := strconv.FormatInt(int64(rand.Float64()*1000000000000000), 10)
		cookie = &http.Cookie{
			Name:    "gsessionid",
			Value:   sess,
			Path:    "/",
			Expires: time.Now().Add(60 * time.Hour),
		}
		http.SetCookie(rw, cookie)
	}
	var cometId string
	var index uint64
	rw.Header().Add("Content-Type", "text/html; charset=UTF-8")
	cometStore.RLock()
	cometVal, found := cometStore.m[session(cookie.Value)]
	cometStore.RUnlock()
	if found {
		cometId = cometVal.Value
	} else {
		//create comet for the first time
		rand.Seed(time.Now().UnixNano())
		cometId = strconv.FormatInt(int64(rand.Float64()*1000000000000000), 10)
		cometStore.Lock()
		cometStore.m[session(cookie.Value)] = comet{cometId, time.Now()}
		cometStore.Unlock()
	}

	messageStore.RLock()
	messages, found := messageStore.m[sessionCometKey(cookie.Value+cometId)]
	lastId := messageStore.LastIndex
	messageStore.RUnlock()
	if found {
		index = lastId
	}

	fmt.Printf("messages %+v", messages)

	err = t.ExecuteTemplate(rw, "index.html", TemplateInfo{
		CometId:  cometId,
		Index:    index,
		Messages: messages})
	if err != nil {
		log.Fatalf("got error: %s", err)
	}

}

func UnixToString(x int64) string {
	ret := time.Unix(x/1000, 0)
	return ret.String()
}

func handleComet(rw http.ResponseWriter, req *http.Request) {
	// session-id}/{page-id // parameters we get
	log.Printf("\n\nNumGoroutine %v\n", runtime.NumGoroutine())
	rw.Header().Set("Content-Type", "application/json")
	currentComet := req.FormValue("cometid")
	currentIndex, _ := strconv.ParseUint(req.FormValue("index"), 10, 64)
	cookie, _ := req.Cookie("gsessionid")
	cometStore.Lock()
	cometStore.m[session(cookie.Value)] = comet{currentComet, time.Now()} //update timestamp on comet
	cometStore.Unlock()
	var chanMessages = make(chan Responses)
	var done = make(chan bool)
	tick := time.NewTicker(500 * 2 * time.Millisecond)
	key := sessionCometKey(cookie.Value + currentComet)
	go func() {
		for {
			select {
			case <-tick.C:
				getMessages(key, currentIndex, chanMessages, done)
			case <-done:
				return
			}
		}
	}()

	select {
	case messages := <-chanMessages:
		done <- true
		json.NewEncoder(rw).Encode(messages)
	case <-time.After(time.Second * 60):
		done <- true
		json.NewEncoder(rw).Encode(Responses{[]Response{Response{Value: jsCmd{""}, Error: ""}}, currentIndex, ""})
	}
}

func getMessages(key sessionCometKey, currentIndex uint64, result chan Responses, done chan bool) {
	messageStore.RLock()
	messages, found := messageStore.m[key]
	lastId := messageStore.LastIndex
	messageStore.RUnlock()
	if found {
		var payload Responses
		for _, msg := range messages {
			if currentIndex < msg.index {
				payload.Res = append(payload.Res, Response{jsCmd{msg.Value.Js}, ""})
			} else {
				log.Printf("not sending message %+v\n", msg)
			}
		}
		if len(payload.Res) > 0 {
			payload.LastIndex = lastId
			payload.Event = "dataMessages"
			result <- payload
		}
	}
}

func gc() {
	for _ = range time.Tick(10 * time.Second) {
		log.Println("Started gc")
		start := time.Now()
		messageStore.Lock()
		for storeKey, messages := range messageStore.m {
			var temp []message
			for key, message := range messages {
				if time.Now().Sub(message.Stamp) > 20*time.Second {
					log.Printf("temp starts as %+v ", temp)
					temp = append(messages[:key], messages[key+1:]...)
					log.Printf("temp ends as %+v ", temp)
				}
			}
			messageStore.m[storeKey] = temp
		}
		messageStore.Unlock()
		log.Printf("Ended gc. It took %v ms\n", time.Now().Sub(start).Nanoseconds()/1000)
	}
}
