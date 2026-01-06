package parallel

import "sync"

// ProcessFiles runs a function on files in parallel with a worker pool.
// The function fn should return (result, include) where include indicates
// whether to include the result in the output slice.
func ProcessFiles[T any](files []string, workers int, fn func(path string) (T, bool)) []T {
	if len(files) == 0 {
		return nil
	}
	if workers <= 0 {
		workers = 4
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]T, 0, len(files))

	fileChan := make(chan string, len(files))
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				if result, include := fn(path); include {
					mu.Lock()
					results = append(results, result)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	return results
}

// ProcessFilesWithLimit runs a function on files in parallel with a limit.
// Stops processing when limit results are collected.
func ProcessFilesWithLimit[T any](files []string, workers, limit int, fn func(path string) (T, bool)) []T {
	if len(files) == 0 {
		return nil
	}
	if workers <= 0 {
		workers = 4
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]T, 0)

	fileChan := make(chan string, len(files))
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				// Check if we've hit the limit
				mu.Lock()
				if limit > 0 && len(results) >= limit {
					mu.Unlock()
					return
				}
				mu.Unlock()

				if result, include := fn(path); include {
					mu.Lock()
					if limit <= 0 || len(results) < limit {
						results = append(results, result)
					}
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}
