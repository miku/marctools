// Create a seekmap of the form (sorted by OFFSET)
// ID OFFSET LENGTH
package main

import (
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "os/exec"
    "strconv"
    "strings"
)

const app_version = "1.3.4"

func record_length(reader io.Reader) (length int64, err error) {
    var l int
    data := make([]byte, 24)
    n, err := reader.Read(data)
    if err != nil {
        return 0, err
    } else {
        if n != 24 {
            errs := fmt.Sprintf("MARC21: invalid leader: expected 24 bytes, read %d", n)
            err = errors.New(errs)
        } else {
            l, err = strconv.Atoi(string(data[0:5]))
            if err != nil {
                errs := fmt.Sprintf("MARC21: invalid record length: %s", err)
                err = errors.New(errs)
            }
        }
    }
    return int64(l), err
}

func main() {

    version := flag.Bool("v", false, "prints current program version")

    var PrintUsage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] MARCFILE\n", os.Args[0])
        flag.PrintDefaults()
    }

    flag.Parse()

    if *version {
        fmt.Println(app_version)
        os.Exit(0)
    }

    if flag.NArg() < 1 {
        PrintUsage()
        os.Exit(1)
    }

    filename := flag.Args()[0]
    fi, err := os.Open(filename)

    if err != nil {
        fmt.Printf("%s\n", err)
        os.Exit(1)
    }

    defer func() {
        if err := fi.Close(); err != nil {
            panic(err)
        }
    }()

    yaz, err := exec.LookPath("yaz-marcdump")
    if err != nil {
        log.Fatal("yaz-marcdump is required")
        os.Exit(1)
    }

    awk, err := exec.LookPath("awk")
    if err != nil {
        log.Fatal("awk is required")
        os.Exit(1)
    }

    cmd := fmt.Sprintf("%s %s | %s ' /^001 / {print $2}'", yaz, filename, awk)
    out, err := exec.Command("bash", "-c", cmd).Output()
    if err != nil {
        log.Fatal(err)
        os.Exit(1)
    }

    ids := strings.Split(string(out), "\n")

    var i, offset int64
    for {
        length, err := record_length(fi)
        if err == io.EOF {
            break
        }
        fmt.Printf("%s\t%d\t%d\n", ids[i], offset, length)
        offset += length
        i += 1
        fi.Seek(offset, 0)
    }
}