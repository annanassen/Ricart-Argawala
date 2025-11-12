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
	sisterNodes []activeNode
}

func (s *messageServer_WOMEN_IN_STEM) sendMessage(proto.MessageServerServer) {

}

func main() {

	nodeA := activeNode{
		ID:          "A",
		portAddress: "5000",
	}
	nodeB := activeNode{
		ID:          "B",
		portAddress: "5001",
	}
	nodeC := activeNode{
		ID:          "C",
		portAddress: "5002",
	}
	nodeA.sisterNodes = append(nodeA.sisterNodes, nodeB, nodeC)
	nodeB.sisterNodes = append(nodeB.sisterNodes, nodeA, nodeC)
	nodeC.sisterNodes = append(nodeC.sisterNodes, nodeA, nodeB)

	MS1 := messageServer_WOMEN_IN_STEM{
		nodes: make([]activeNode, 0),
		clock: 0,
	}

	MS2 := messageServer_WOMEN_IN_STEM{
		nodes: make([]activeNode, 0),
		clock: 0,
	}

	MS3 := messageServer_WOMEN_IN_STEM{
		nodes: make([]activeNode, 0),
		clock: 0,
	}

	MS1.nodes = append(MS1.nodes, nodeB, nodeC)
	MS2.nodes = append(MS2.nodes, nodeA, nodeC)
	MS3.nodes = append(MS3.nodes, nodeA, nodeB)

	go start_servers(nodeA, &MS1)
	go start_servers(nodeB, &MS2)
	go start_servers(nodeC, &MS3)

	connectAnode(nodeA)
	connectAnode(nodeB)
	connectAnode(nodeC)

}

func connectAnode(node activeNode) {

	for _, sisterNode := range node.sisterNodes {
		conn, err := grpc.NewClient("localhost:"+sisterNode.portAddress, grpc.WithTransportCredentials(insecure.NewCredentials())) // I want to make a connection pls
		if err != nil {                                                                                                            // If an error occurs:
			log.Fatalf("Not working")
		}

		fmt.Println("Node", node.ID, "is connected to node", sisterNode.ID, "on port", sisterNode.portAddress)

		client := proto.NewMessageServerClient(conn) // Creates a client object
		// Knows how to send and receive messages now.
		fmt.Println(client, "YAYY")
	}

}

func start_servers(activeNode activeNode, MS *messageServer_WOMEN_IN_STEM) {
	fmt.Println("Node", activeNode.ID, "started server on port", activeNode.portAddress)
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
