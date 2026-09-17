package tasks

type Store interface {
	GetAll() []Task
	GetByID(id int) (Task, error)
	Create(task Task) (Task, error)
	Update(id int, task Task) (Task, error)
	Delete(id int) error
}
