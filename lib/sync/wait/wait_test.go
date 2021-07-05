package wait

import "testing"

func TestWait(t *testing.T) {
	w := Wait{}
	w.Add(3)
	w.Wait()
	w.Done()
}
