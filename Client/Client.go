package client

type Node struct {
}

func Bid(amount int) {
	// call to server brr brr, we bid this amount
}

func Query() {
	// query system to find amount
	return 0
}

// Comment: given a bid, returns an outcome among {fail, success or exception}

func main() {
	// while server open
	highest_bid := Query()

	Bid(highest_bid + 1)
}

func Result() {
	//
}

// Comment: if the auction is over, it returns the result, else highest bid.
