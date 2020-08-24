package main

import (
	"os"
	"time"
	"encoding/json"
	"net/http"
	"fmt"
	"strings"
	"net"
)

var a2sinfo = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x54, 0x53, 0x6F, 0x75, 0x72, 0x63, 0x65, 0x20, 0x45, 0x6E, 0x67, 0x69, 0x6E, 0x65, 0x20, 0x51, 0x75, 0x65, 0x72, 0x79, 0x00}

func main(){
	inithttpserver()
}

func inithttpserver(){
	http.HandleFunc("/", urlsplitter)
	http.HandleFunc("/prometheus/", urlsplitter)
	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	http.ListenAndServe(":"+port, nil)
}

func urlsplitter(w http.ResponseWriter, req *http.Request) {
	p := strings.Split(req.URL.Path, "/")
	if len(p) == 2 {
		fmt.Fprintf(w,"You need to enter the URL like this: https://example.com/192.168.0.2/27015 if the IP of the CS:GO Server is running on the IP 192.168.0.2 and the server uses the default port 27015.")
	} else if len(p) == 3  || len(p) == 4 && strings.ToLower(p[1]) == "prometheus"{
		if strings.ToLower(p[1]) == "prometheus" {
			sendData(p[2], p[3], w, true)
		} else {
			sendData(p[1], p[2], w, false)
		}
	} else {
		fmt.Fprintf(w,"Wrong input.")
	}
}

func sendData(ip string, port string, w http.ResponseWriter, prometheus bool) {
	//Listener
	Listener, err := net.ListenPacket("udp", ":0")
	if err != nil {
		errorinfo, _ := json.Marshal(errorinformation{Error: err.Error()})
		defer Listener.Close()
		w.Write(errorinfo)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer Listener.Close()
	//Resolve port and ip into an Address
	RemoteAddr, err := net.ResolveUDPAddr("udp", ip+":"+port)
	if err != nil {
		errorinfo, _ := json.Marshal(errorinformation{Error: err.Error()})
		w.Write(errorinfo)
		return
	}
	Listener.SetDeadline(time.Now().Add(time.Second*5))
	_,err = Listener.WriteTo(a2sinfo, RemoteAddr)
	if err != nil {
		errorinfo, _ := json.Marshal(errorinformation{Error: err.Error()})
		w.Write(errorinfo)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	buf := make([]byte, 256)
	n, _, err := Listener.ReadFrom(buf)
	_ = n
	if err != nil {
		errorinfo, _ := json.Marshal(errorinformation{Error: err.Error()})
		w.Write(errorinfo)
		return
	}
	//Format information
	counter := 6
	serverinfo := string(buf[counter:len(buf)-1])
	servername, index := substring(serverinfo)
	counter += index+1
	mapname, index := substring(string(buf[counter:len(buf)-1]))
	counter += index+1
	foldername, index := substring(string(buf[counter:len(buf)-1]))
	counter += index+1
	gamename, index := substring(string(buf[counter:len(buf)-1]))
	//+2 to remove 16 Bit SteamApplicationId
	counter += index+3
	if prometheus {
		characters := []string{"{", "}", "[", "]", ",", "."}
		for _, character := range characters {
			servername = strings.Replace(servername, character, "", len(servername))
		}
		servername = strings.Replace(servername, " ", "_", len(servername))
		servername = strings.ToLower(servername)
		fmt.Fprintf(w, "src_"+servername+"_status 1\nsrc_"+servername+"_players %v\nsrc_"+servername+"_botplayer %v",string(buf[counter:len(buf)-1])[0], string(buf[counter:len(buf)-1])[2])
	} else {
		server, _ := json.Marshal(serverinformation{Servername: servername, Map: mapname, Foldername: foldername, Game: gamename, PlayerCount: string(buf[counter:len(buf)-1])[0], MaxPlayerCount: string(buf[counter:len(buf)-1])[1], BotPlayerCount: string(buf[counter:len(buf)-1])[2]})
		w.Write(server)
	}
	defer Listener.Close()
}

func substring(content string) (string, int){
	placeholder := ""
	i := 0
	for i = 0; i < len(content); i++ {
		if content[i] == 0 {
			break
		}
		placeholder = placeholder+string(content[i])
	}
	return placeholder, i
}

type serverinformation struct {
	Servername string;
	Game string;
	Map string;
	PlayerCount byte;
	MaxPlayerCount byte;
	BotPlayerCount byte;
	Foldername string;
}

type errorinformation struct {
	Error string;
}