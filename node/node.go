package main

import (
	proto "ITU/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type messageServer_WOMEN_IN_STEM struct {
	proto.UnimplementedMessageServerServer
	mu           sync.RWMutex //mutex prevents race conditions
	portAddress  string // The port
	nodes        map[string]*grpc.ClientConn // Ports mapped to their connection
	critical     bool // Is it in the critical section (Freja is correct)
	requesting   bool // Requesting access to critical section
	timestamp    int64 // Lamport time yes
	reqTimestamp int64  // Lamport time when request was made
	theDMs       []string // This actually means deferred responses (idk spørg Anna)
	replies      int // How mny many nodes have granted access to critical zone
}

func (serve *messageServer_WOMEN_IN_STEM) SendRequest(ctx context.Context, req *proto.Req) (*proto.Empty, error) {
	serve.mu.Lock()
	myTS := serve.reqTimestamp
	theirTS := req.Timestamp
	serve.timestamp = max(serve.timestamp, req.Timestamp) + 1
	serve.mu.Unlock()
	if serve.critical || (serve.requesting && myTS <= theirTS) {
		log.Printf("Deferring %s access to critical section (%d<%d)\n", req.Port, myTS, theirTS)
		serve.theDMs = append(serve.theDMs, req.Port)
		return &proto.Empty{}, nil
	}

	c := proto.NewMessageServerClient(serve.nodes[req.Port])
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serve.mu.Lock()
	serve.timestamp++
	TS := serve.timestamp
	serve.mu.Unlock()
	log.Printf("Allowing %s access to critical section\n", req.Port)
	c.SendResponse(ctx, &proto.Resp{Timestamp: TS, Port: serve.portAddress})
	return &proto.Empty{}, nil
}

func (serve *messageServer_WOMEN_IN_STEM) SendResponse(ctx context.Context, resp *proto.Resp) (*proto.Empty, error) {
	serve.mu.Lock()     
	serve.timestamp = max(serve.timestamp, resp.Timestamp) + 1
	serve.mu.Unlock()
	
	log.Println(resp.Port, "allowed accesss to critical section")
	serve.replies++

	return &proto.Empty{}, nil
}

func main() {
	start_server()
}

func (serve *messageServer_WOMEN_IN_STEM) get_available_listener() net.Listener {
	// Private port range
	var ports []string
	ports = append(ports, ":50000")
	ports = append(ports, ":50001")
	ports = append(ports, ":50002")

	var ret net.Listener

	for _, port := range ports {
		if ret == nil {
			listener, err := net.Listen("tcp", port)
			if err == nil {
				log.Println("Got port", port)
				serve.portAddress = port
				ret = listener
				continue
			}
		}
		
		serve.nodes[port] = nil
	}
	if ret == nil {
		log.Fatalf("Could not eat") 
	}

	return ret
}

func (serve *messageServer_WOMEN_IN_STEM) exit() {
	if !serve.critical {
		log.Printf("Not in critical zone")
		return
	} 

	serve.critical = false

	serve.mu.Lock()
	serve.timestamp++
	TS := serve.timestamp
	serve.mu.Unlock()

	for _, port := range serve.theDMs {
		func () {
			c := proto.NewMessageServerClient(serve.nodes[port])
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			log.Printf("Allowing deferred %s access to critical section\n", port)
			c.SendResponse(ctx, &proto.Resp{Timestamp: TS, Port: serve.portAddress})
		}()
	}
}

func (serve *messageServer_WOMEN_IN_STEM) enter() {
	if serve.critical {
		log.Printf("Already has access to critical section")
		return
	} else if serve.requesting {
		log.Printf("Already requesting access to critical section")
		return
	}

	serve.requesting = true
	serve.mu.Lock()
	serve.timestamp++
	serve.reqTimestamp = serve.timestamp
	serve.mu.Unlock()
	for port, conn := range serve.nodes {
		c := proto.NewMessageServerClient(conn)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { // Func to defer cancel()
			defer cancel()
			c.SendRequest(ctx, &proto.Req{Timestamp: serve.reqTimestamp, Port: serve.portAddress})
		}()
		log.Println("Requesting access to critical section from", port)
	}


	requiredReplies := 2
	for {
		if requiredReplies == serve.replies {
			break
		}
	}
	serve.critical = true
	serve.requesting = false

	log.Printf("Gained access to critical section")
	serve.replies = 0
}

func (serve *messageServer_WOMEN_IN_STEM) connect_nodes() {
	for port, _ := range serve.nodes {
		conn, err := grpc.NewClient(port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to slay")
		}
		serve.nodes[port] = conn
	}
}

func start_server() {
	node := grpc.NewServer()
	serve := &messageServer_WOMEN_IN_STEM{
		nodes: make(map[string]*grpc.ClientConn),
	}
	listener := serve.get_available_listener()

	proto.RegisterMessageServerServer(node, serve)

	go func() {
		if err := node.Serve(listener); err != nil {
			log.Fatalf("Failed to gatekeep")
		}
	}()

	fmt.Println("Once all nodes have been started type MS")
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		if sc.Text() == "MS" {
			serve.connect_nodes()
			break
		}
	}

	fmt.Println("type:")
	fmt.Println("    enter: enter the critical section")
	fmt.Println("    exit:  leave the critical section")
	for sc.Scan() {
		action := sc.Text()
		switch action {
		case "enter":
			go serve.enter()
		case "exit":
			go serve.exit()
		}
	}
}