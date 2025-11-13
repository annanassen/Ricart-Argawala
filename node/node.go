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
	portAddress  string
	nodes        map[string]*grpc.ClientConn
	critical     bool
	requesting   bool
	timestamp    int64
	reqTimestamp int64
	theDMs       []string
	replies      int
}

func (serve *messageServer_WOMEN_IN_STEM) SendRequest(ctx context.Context, req *proto.Req) (*proto.Empty, error) {
	log.Printf("Fcuk")

	serve.mu.Lock()
	myTS := serve.reqTimestamp
	theirTS := req.Timestamp
	serve.timestamp = max(serve.timestamp, req.Timestamp)
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
	c.SendResponse(ctx, &proto.Resp{Timestamp: serve.timestamp, Port: serve.portAddress})
	serve.mu.Unlock()
	log.Printf("Allowing %s access to critical section\n", req.Port)
	return &proto.Empty{}, nil
}

func (serve *messageServer_WOMEN_IN_STEM) SendReply(ctx context.Context, resp *proto.Resp) (*proto.Empty, error) {
	serve.mu.Lock()     
	defer serve.mu.Unlock() 
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
		if ret == nil{
			listener, err := net.Listen("tcp", port)
			if err == nil {
				log.Println("Got port", port)
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

func (serve *messageServer_WOMEN_IN_STEM) request() {
	if serve.critical {
		log.Printf("Already has access to critical section")
		return
	} else if serve.requesting {
		log.Printf("Already requesting access to critical section")
		return
	}
	serve.mu.Lock()
	serve.timestamp++
	serve.reqTimestamp = serve.timestamp
	serve.mu.Unlock()
	go func() { // Func to defer cancel()
		for port, conn := range serve.nodes {
			c := proto.NewMessageServerClient(conn)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			log.Println("Requesting access to critical section from", port)
			go c.SendRequest(ctx, &proto.Req{Timestamp: serve.reqTimestamp, Port: serve.portAddress})
		}
	}()

	requiredReplies := 2
	for {
		if requiredReplies == serve.replies {
			break
		}
	}

	log.Printf("Gained access to critical section")
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

	serve.portAddress = listener.Addr().String()[len(listener.Addr().String())-6:] // port number

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
	fmt.Println("    leave: leave the critical section")
	for sc.Scan() {
		action := sc.Text()
		switch action {
		case "enter":
			{
				go serve.request()
			}
		case "leave":
			{
				// TODO: leave
			}
		}
	}
}
