package main

import (
    "io/ioutil"
    "log"
    "path/filepath"
    "os"
    "os/exec"
    "runtime"
)

func main() {
    _, filename, _, _ := runtime.Caller(0)
    root := filepath.Dir(filename)
    dirs, err := ioutil.ReadDir(root)
    if err != nil {
        log.Fatal(err)
    }

    for _, d := range dirs {
        testFile := filepath.Join(root, d.Name(), "test.go")
        if _, err := os.Stat(testFile); err == nil {
            cmd := exec.Command("go", "run", testFile)
            log.Printf("Running command: go run %s", testFile)
            output, err := cmd.CombinedOutput()
            log.Printf("%s", output)
            if err != nil {
                log.Fatalf("Test failed for project %s", d.Name())
            }
            log.Printf("Test finished successfully for project %s", d.Name())
        }
    }
}