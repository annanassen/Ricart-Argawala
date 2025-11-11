package main

import (
	proto "ITU/grpc"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type messageServer_WOMEN_IN_STEM struct {
	proto.UnimplementedMessageServerServer
	mu    sync.RWMutex //mutex prevents race conditions
	nodes []activeNode //(pizza)slice that keeps track of the clients on the server
	clock int64        //logical clock (lamport)
}

type activeNode struct {
	ID          string
	portAddress string
	addressList []string
	timeStamp   int64
}

func (s *messageServer_WOMEN_IN_STEM) sendMessage(proto.MessageServerServer) {

}

func main() {

	nodeA := activeNode{
		ID:          "A",
		portAddress: "5000",
		addressList: []string{"5001", "5002"},
		timeStamp:   0,
	}
	nodeB := activeNode{
		ID:          "B",
		portAddress: "5001",
		addressList: []string{"5000", "5002"},
		timeStamp:   0,
	}
	nodeC := activeNode{
		ID:          "C",
		portAddress: "5002",
		addressList: []string{"5000", "5001"},
		timeStamp:   0,
	}

	MS := messageServer_WOMEN_IN_STEM{
		nodes: make([]activeNode, 0),
		clock: 0,
	}

	MS.mu.Lock() // Locks
	MS.nodes = append(MS.nodes, nodeA, nodeB, nodeC)
	MS.mu.Unlock()

	go start_servers(nodeA, &MS)
	go start_servers(nodeB, &MS)
	go start_servers(nodeC, &MS)

	connectAnode(nodeA)
	connectAnode(nodeB)
	connectAnode(nodeC)

}

func connectAnode(node activeNode) {

	for _, sisterPort := range node.addressList {
		conn, err := grpc.NewClient("localhost:"+sisterPort, grpc.WithTransportCredentials(insecure.NewCredentials())) // I want to make a connection pls
		if err != nil {                                                                                                // If an error occurs:
			log.Fatalf("Not working")
		}

		fmt.Println("Node ", node.ID, "is connected to node on port ", sisterPort)

		client := proto.NewMessageServerClient(conn) // Creates a client object
		// Knows how to send and receive messages now.
		fmt.Println(client, "YAYY")
	}

}

func start_servers(activeNode activeNode, MS *messageServer_WOMEN_IN_STEM) {
	fmt.Println("Node ", activeNode.ID, "started server on port", activeNode.portAddress)
	node := grpc.NewServer()

	listener, err := net.Listen("tcp", ":"+activeNode.portAddress)
	if err != nil {
		log.Fatalf("Did not work")
	}

	designatedNode := MS
	proto.RegisterMessageServerServer(node, designatedNode)

	err = node.Serve(listener)
	if err != nil {
		log.Fatalf("Did not serve")
	}
}
