package task

var TaskPool Pool

type Pool interface {
	Add(num int)
	Submit(job Job)
}
