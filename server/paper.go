package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	paper "paper/database"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PaperServer struct {
	papers []paper.Paper
	nextID int
	conn   *amqp.Connection
	ch     *amqp.Channel
	sync.RWMutex
}

func (s *PaperServer) AddPaper(args paper.AddPaperArgs, reply *paper.AddPaperReply) error {
	s.Lock()
	defer s.Unlock()
	s.nextID++
	newPaper := paper.Paper{PaperNumber: s.nextID, Author: args.Author, Title: args.Title, Format: args.Format, Content: args.Content}
	s.papers = append(s.papers, newPaper)

	go func() {
		err := s.ch.Publish(
			"paper_events",
			"",
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(fmt.Sprintf("\nNew paper added: id: %d Author: %s Title: %s\nEnter command (add, list, detail, fetch, exit): ", s.nextID, args.Author, args.Title)),
			},
		)
		if err != nil {
			log.Printf("Error publishing to RabbitMQ: %v", err)
		}
	}()

	reply.PaperNumber = s.nextID
	return nil
}

func (s *PaperServer) ListPapers(args paper.ListPapersArgs, reply *paper.ListPapersReply) error {
	s.RLock()
	defer s.RUnlock()
	reply.Papers = s.papers
	return nil
}

func (s *PaperServer) GetPaperDetails(args paper.GetPaperArgs, reply *paper.GetPaperDetailsReply) error {
	s.RLock()
	defer s.RUnlock()
	for _, p := range s.papers {
		if p.PaperNumber == args.PaperNumber {
			reply.Author = p.Author
			reply.Title = p.Title
			return nil
		}
	}
	reply.Error = "Paper not found"
	return nil
}

func (s *PaperServer) FetchPaperContent(args paper.FetchPaperArgs, reply *paper.FetchPaperReply) error {
	s.RLock()
	defer s.RUnlock()
	for _, p := range s.papers {
		if p.PaperNumber == args.PaperNumber {
			reply.Content = p.Content
			return nil
		}
	}
	reply.Error = "Paper not found"
	return nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"paper_events",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a fanout exchange")

	server := &PaperServer{papers: make([]paper.Paper, 0), nextID: 0, conn: conn, ch: ch}
	rpc.Register(server)
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	fmt.Println("Server listening on :8080")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		go jsonrpc.ServeConn(conn)
	}
}
