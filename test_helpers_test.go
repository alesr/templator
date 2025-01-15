package templator

// Temporary implementations for testing - simulating generated code
func (r *Registry[T]) GetHome() (*Handler[T], error) {
	return r.Get("home")
}

func (r *Registry[T]) GetComponentsMenu() (*Handler[T], error) {
	return r.Get("components/menu")
}
