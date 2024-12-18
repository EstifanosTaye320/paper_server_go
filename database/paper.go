package paper

import (
	"bytes"
	"encoding/gob"
)

type Paper struct {
	PaperNumber int
	Author      string
	Title       string
	Format      string
	Content     []byte
}

type AddPaperArgs struct {
	Author  string
	Title   string
	Format  string
	Content []byte
}

type AddPaperReply struct {
	PaperNumber int
	Error       string
}

type ListPapersArgs struct{}

type ListPapersReply struct {
	Papers []Paper
	Error  string
}

type GetPaperArgs struct {
	PaperNumber int
}

type GetPaperDetailsReply struct {
	Author string
	Title  string
	Error  string
}

type FetchPaperArgs struct {
	PaperNumber int
}

type FetchPaperReply struct {
	Content []byte
	Error   string
}

func EncodePaper(paper Paper) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(paper)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodePaper(data []byte) (Paper, error) {
	var paper Paper
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&paper)
	if err != nil {
		return paper, err
	}
	return paper, nil
}
