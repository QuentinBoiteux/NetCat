package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	connections   []net.Conn
	oldMessages   []string
	connectionsMu sync.Mutex          // Mutex pour protéger la slice connections
	groups        map[net.Conn]string // Map qui associe chaque connexion à un groupe
)

func main() {
	var port int = 8989
	// Vérifie si un argument a été fourni

	args := os.Args[1:]
	if len(args) == 0 {
	} else if len(args) == 1 {
		portInt, _ := strconv.Atoi(args[0])
		port = portInt
	} else {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}

	// obtient l'adresse IP courante
	IP, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	// Convertir pour l'utiliser comme une valeur pour 'net.Parse'
	IPstr := []string(nil)
	IPstr = func(addrs []net.Addr) []string {
		for _, addr := range addrs {
			IPstr = append(IPstr, addr.String())
		}
		return IPstr
	}(IP)
	// Parser en tant que type IP, '[1]' car c'est une chaîne de caractères et l'IP est à l'index 1
	addrr := &net.TCPAddr{IP: net.ParseIP(IPstr[1]), Port: port}
	fmt.Println("nc", IPstr[1], addrr)
	// Socket en écoute
	ln, err := net.Listen("tcp", addrr.String())
	if err != nil {
		panic(err)
	}
	// Ferme le socket lorsque le serveur sera fermé
	defer ln.Close()
	fmt.Println("Listening on the port :", port)
	// Initialize the groups map
	groups = make(map[net.Conn]string)
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn)
	}
}

// Envoyer un message à tous les autres clients du même groupe
func sendMessage(message string, ignore net.Conn) {
	// Obtient la date et l'heure actuelles
	now := time.Now()
	// Formate la date et l'heure en tant que chaîne de caractères
	dateTimeStr := now.Format("2006-01-02 15:04:05")

	// Prepend the date and time to the message
	message = "[" + dateTimeStr + "]" + " " + message
	// Add the message to the list of old messages
	oldMessages = append(oldMessages, message)
	// Open the log file in append mode
	logFile, err := os.OpenFile("chat_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Println(err)
		return
	}
	defer logFile.Close()
	// Write the message to the log file
	_, err = logFile.WriteString(message)
	if err != nil {
		log.Println(err)
		return
	}
	// Get the group of the sender
	senderGroup, ok := groups[ignore]
	if !ok {
		// The sender is not in a group, so don't send the message to anyone
		return
	}
	// Lock the connections slice
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	// Iterate through the list of connections
	for _, conn := range connections {
		// Don't send the message to the client that is leaving
		if conn != ignore {
			// Check if the connection belongs to the same group as the sender
			recipientGroup, ok := groups[conn]
			if !ok {
				// The recipient is not in a group, so don't send the message
				continue
			}
			if recipientGroup == senderGroup {
				// The recipient is in the same group as the sender, so send the message
				fmt.Fprint(conn, message)
			}
		}
	}
}

// Handle incoming connections
func handleConnection(conn net.Conn) {
	// Check if the number of connections is already at the maximum
	if len(connections) >= 10 {
		fmt.Fprintln(conn, "The chat room is full. Please try again later.")
		conn.Close()
		return
	}
	// Add the connection to the list
	connectionsMu.Lock()
	connections = append(connections, conn)
	connectionsMu.Unlock()
	SendWelcome(conn)
	// Display old messages to the new user
	// Use bufio to read from the connection
	reader := bufio.NewReader(conn)
	// Read the user's nickname
	var nickname string
	for {
		fmt.Fprint(conn, "[ ENTER YOUR NAME ]: ")
		nickname, _ = reader.ReadString('\n')
		nickname = strings.TrimSpace(nickname)
		if len(nickname) > 0 {
			break
		}
	}
	// Read the user's group
	var group string
	for {
		fmt.Fprint(conn, "[ ENTER YOUR GROUP ]: ")
		group, _ = reader.ReadString('\n')
		group = strings.TrimSpace(group)
		if len(group) > 0 {
			break
		}
	}
	// Associate the connection with the group
	groups[conn] = group
	// Send a message to all other clients to announce the new user
	sendMessage(fmt.Sprintf("[%s] has joined the chat!\n", nickname), conn)
	for _, oldMessage := range oldMessages {
		fmt.Fprint(conn, oldMessage)
	}
	for {
		// Read the user's input
		fmt.Fprint(conn, "> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Client disconnected")
			} else {
				fmt.Println("Error reading input:", err)
			}
			break
		}
		// Trim leading and trailing white space
		input = strings.TrimSpace(input)
		// Check if the user is leaving the chat
		if strings.ToLower(input) == "leave" {
			break
		}
		// Create a buffer to write the message to
		var buffer bytes.Buffer
		// Write the message to the buffer
		buffer.WriteString("[")
		buffer.WriteString(nickname)
		buffer.WriteString("] ")
		buffer.WriteString(input)
		buffer.WriteString("\n")
		// Send the message to all other clients
		sendMessage(buffer.String(), conn)
	}
	// Remove the connection from the list
	connectionsMu.Lock()
	for i, c := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
			break
		}
	}
	connectionsMu.Unlock()
	// Send a message to all other clients to announce that the user is leaving
	sendMessage(fmt.Sprintf("[%s] has left the chat.\n", nickname), conn)
	// Close the connection
	conn.Close()
}

// Send a welcome message to a client
func SendWelcome(conn net.Conn) {
	// Open the text file
	file, err := os.Open("Pingu.txt")
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	// Create a buffered reader to read from the file
	reader := bufio.NewReader(file)
	// Read the file line by line
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			// Handle the error
			log.Println(err)
		}
		// Send the line to the client
		fmt.Fprint(conn, line)
	}
	fmt.Fprint(conn, "To leave the chat, type 'leave'\n")
	fmt.Fprint(conn, "\n")
}
