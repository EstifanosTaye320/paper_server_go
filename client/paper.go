package main

import (
	"bufio"
	"fmt"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	paper "paper/database"
	"strconv"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

func addPaper(client *rpc.Client, parts []string) {
	if len(parts) < 4 {
		fmt.Println("Usage: add <author> <title> <file>")
		return
	}
	author := parts[1]
	title := parts[2]
	filename := parts[3]
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("Error reading file:", err)
	}
	args := paper.AddPaperArgs{Author: author, Title: title, Format: filename[len(filename)-3:], Content: content}
	var reply paper.AddPaperReply
	err = client.Call("PaperServer.AddPaper", args, &reply)
	if err != nil {
		log.Fatal("AddPaper error:", err)
	}
	fmt.Printf("Paper added with ID: %d\n", reply.PaperNumber)
}

func listPapers(client *rpc.Client) {
	var reply paper.ListPapersReply
	err := client.Call("PaperServer.ListPapers", paper.ListPapersArgs{}, &reply)
	if err != nil {
		log.Fatal("ListPapers error:", err)
	}
	fmt.Println("Papers:")
	for _, p := range reply.Papers {
		fmt.Printf("ID: %d, Author: %s, Title: %s\n", p.PaperNumber, p.Author, p.Title)
	}
}

func getPaperDetails(client *rpc.Client, parts []string) {
	if len(parts) != 2 {
		fmt.Println("Usage: detail <paper_number>")
		return
	}
	paperNumber, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Fatal("Invalid paper number")
	}
	args := paper.GetPaperArgs{PaperNumber: paperNumber}
	var reply paper.GetPaperDetailsReply
	err = client.Call("PaperServer.GetPaperDetails", args, &reply)
	if err != nil {
		log.Fatal("GetPaperDetails error:", err)
	}
	fmt.Printf("Author: %s, Title: %s\n", reply.Author, reply.Title)
}

func fetchPaperContent(client *rpc.Client, parts []string) {
	if len(parts) != 2 {
		fmt.Println("Usage: fetch <paper_number>")
		return
	}
	paperNumber, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Fatal("Invalid paper number")
	}
	args := paper.FetchPaperArgs{PaperNumber: paperNumber}
	var reply paper.FetchPaperReply
	err = client.Call("PaperServer.FetchPaperContent", args, &reply)
	if err != nil {
		log.Fatal("FetchPaperContent error:", err)
	}
	fmt.Println(string(reply.Content))
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	serverAddress := "localhost:8080"

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"",
		false,
		true,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.QueueBind(
		q.Name,
		"",
		"paper_events",
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to consume a queue")

	client, err := jsonrpc.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatal("Error connecting to server:", err)
	}
	defer client.Close()

	go func() {
		for d := range msgs {
			fmt.Printf("%s", d.Body)
		}
	}()

	for {
		fmt.Print("Enter command (add, list, detail, fetch, exit): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		if command == "exit" {
			break
		}

		switch command {
		case "add":
			addPaper(client, parts)
		case "list":
			listPapers(client)
		case "detail":
			getPaperDetails(client, parts)
		case "fetch":
			fetchPaperContent(client, parts)
		default:
			fmt.Println("Invalid command")
		}
	}
}
