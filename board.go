package main

import (
	"fmt"
	"strings"
)

type post struct {
	user string
	text string
}

type board struct {
	children map[string]*board
	canPost  bool
	posts    []post
}

var messageBoards board

func MakeBoard(canPost bool) *board {
	result := board{
		children: make(map[string]*board),
	}
	result.canPost = canPost
	return &result
}

func initializeBoards() {
	messageBoards.children = make(map[string]*board)
}

func createBoard(path string) {
	trimmed := strings.TrimSuffix(path, "/")
	splitPath := strings.Split(trimmed, "/")
	currentBoard := &messageBoards
	for _, s := range splitPath {
		board, ok := currentBoard.children[s]
		if !ok {
			currentBoard.children[s] = MakeBoard(true)
			board = currentBoard.children[s]
		}
		currentBoard = board
	}
}

func getBoardAtPath(path string) (*board, error) {
	boardString := strings.TrimSuffix(path, "/")
	currentBoard := &messageBoards
	if len(boardString) > 0 {
		boardPath := strings.Split(boardString, "/")
		for _, s := range boardPath {
			b, ok := currentBoard.children[s]
			if !ok {
				return nil, fmt.Errorf("Board not found: %v", path)
			}
			currentBoard = b
		}
	}
	return currentBoard, nil
}

func postMessage(path, user, message string) error {
	board, err := getBoardAtPath(path)
	if err != nil {
		return err
	}
	if !board.canPost {
		return fmt.Errorf("Posting on %v is disabled", path)
	}
	board.posts = append(board.posts, post{
		user: user,
		text: message,
	})
	return nil
}

func getMessages(path string) ([]post, error) {
	board, err := getBoardAtPath(path)
	if err != nil {
		return nil, err
	}
	if !board.canPost && len(board.posts) == 0 {
		return nil, fmt.Errorf("Posts are disabled on %v", path)
	}
	result := []post{}
	result = append(result, board.posts...)
	return result, nil
}

func getChildrenBoards(path string) ([]string, error) {
	board, err := getBoardAtPath(path)
	if err != nil {
		return nil, err
	}

	result := []string{}
	for key := range board.children {
		result = append(result, key)
	}

	return result, nil
}
