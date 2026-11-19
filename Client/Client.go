package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pb "auction/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	name := flag.String("name", "", "bidder name")
	nodes := flag.String("nodes", "5001,5002,5003", "ports of all nodes")
	flag.Parse()

	if *name == "" {
		log.Fatal("usage: go run ./Client -name <your name>")
	}

	var clients []pb.AuctionClient
	for _, p := range strings.Split(*nodes, ",") {
		conn, err := grpc.NewClient("localhost:"+p, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatal(err)
		}
		clients = append(clients, pb.NewAuctionClient(conn))
	}

	fmt.Println("Commands: bid <amount> | result")
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		switch {
		case len(f) == 2 && f[0] == "bid":
			amount, err := strconv.ParseInt(f[1], 10, 64)
			if err != nil || amount <= 0 {
				fmt.Println("amount must be a positive integer")
				continue
			}
			fmt.Println(bid(clients, *name, amount))
		case len(f) == 1 && f[0] == "result":
			printResult(clients)
		default:
			fmt.Println("Commands: bid <amount> | result")
		}
	}
}

func bid(clients []pb.AuctionClient, name string, amount int64) pb.BidReply_Outcome {
	for attempt := 0; attempt < 5; attempt++ {
		for _, c := range clients {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			r, err := c.Bid(ctx, &pb.BidRequest{Bidder: name, Amount: amount})
			cancel()
			if err == nil && r.Outcome != pb.BidReply_EXCEPTION {
				return r.Outcome
			}
		}
		time.Sleep(time.Second)
	}
	return pb.BidReply_EXCEPTION
}

func printResult(clients []pb.AuctionClient) {
	for attempt := 0; attempt < 5; attempt++ {
		for _, c := range clients {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			r, err := c.Result(ctx, &pb.Empty{})
			cancel()
			if err != nil {
				continue
			}

			switch {
			case r.GetWinner() == "":
				fmt.Println("no bids yet")
			case r.GetOver():
				fmt.Printf("Auction is over. Winner: %s with %d\n", r.GetWinner(), r.GetAmount())
			default:
				fmt.Printf("Highest bid: %d by %s\n", r.GetAmount(), r.GetWinner())
			}
			return
		}
		time.Sleep(time.Second)
	}
	fmt.Println("no nodes are responding")
}
