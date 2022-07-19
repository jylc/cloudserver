package balancer

type Balancer interface {
	NextPeer(nodes interface{}) (error, interface{})
}

func NewBalancer(strategy string) Balancer {
	switch strategy {
	case "RoundRobin":
		return &RoundRobin{}
	default:
		return &RoundRobin{}
	}
}
