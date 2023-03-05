package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/toxyl/glog"
)

var (
	log = glog.NewLoggerSimple("downloader")
)

func storeCursor() {
	fmt.Print(glog.StoreCursor())
}

func restoreCursor() {
	fmt.Print(glog.RestoreCursor())
}

func clearLine() {
	fmt.Print(glog.RestoreCursor() + "\033[G" + glog.ClearToEOL())
}

func ask(message string) bool {
	message += " [y|N] " + glog.StoreCursor()
	log.Question(message)
	restoreCursor()
	var response string
	_, err := fmt.Scanln(&response)
	ok := false
	if err == nil {
		switch strings.ToLower(response) {
		case "y", "yes":
			ok = true
		case "n", "no":
			ok = false
		default:
			ok = false
		}
	}
	clearLine()
	return ok
}

func downloadFile(
	srcURL string,
	dstDir string,
	onProgress func(fileName string, bytesTotal, bytesRead int64, progress, speed, secondsRemaining float64),
	onProgressSuccess func(fileName string, bytesTotal, bytesRead int64, progress, speed, secondsRemaining float64),
	onProgressError func(fileName string, bytesTotal, bytesRead int64, progress, speed, secondsRemaining float64, err error),
) (string, error) {
	u, err := url.Parse(srcURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		return "", fmt.Errorf("not a valid URL")
	}

	dstName := filepath.Base(u.Path)

	dstPath := filepath.Join(dstDir, dstName)
	if _, err := os.Stat(dstPath); err == nil {
		ok := ask("The file " + glog.File(dstPath) + " already exists, do you want to download a fresh copy?")

		if !ok {
			return dstPath, fmt.Errorf("Download cancelled!")
		}

		err = os.RemoveAll(dstPath)
		if err != nil {
			return dstPath, err
		}
	}

	resp, err := http.Get(srcURL)
	if err != nil {
		return dstPath, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return dstPath, fmt.Errorf("Download failed, received status code %s", glog.Int(resp.StatusCode))
	}

	destFile, err := os.Create(dstPath)
	if err != nil {
		return dstPath, err
	}
	defer destFile.Close()

	bytesTotal := resp.ContentLength
	bytesRead := int64(0)
	chunkBuffer := make([]byte, 1024*1024)
	timeStart := time.Now()
	progress := 0.0
	speed := 0.0
	secondsRemaining := 0.0

	for {
		n, err := resp.Body.Read(chunkBuffer)
		if err != nil && err != io.EOF {
			onProgressError(dstName, bytesTotal, bytesRead, progress, speed, secondsRemaining, err)
			return dstPath, err
		}

		if n > 0 {
			_, err := destFile.Write(chunkBuffer[:n])
			if err != nil {
				onProgressError(dstName, bytesTotal, bytesRead, progress, speed, secondsRemaining, err)
				return dstPath, err
			}
			bytesRead += int64(n)
			progress = float64(bytesRead) / float64(bytesTotal)
			speed = float64(bytesRead) / float64(glog.Max(1, int64(time.Since(timeStart).Seconds())))
			secondsRemaining = float64(bytesTotal-bytesRead) / speed

			onProgress(dstName, bytesTotal, bytesRead, progress, speed, secondsRemaining)
		}

		if err == io.EOF {
			break
		}
	}
	onProgressSuccess(dstPath, bytesTotal, bytesRead, progress, speed, secondsRemaining)
	return dstPath, nil
}

func downloadWithProgress(srcURL string, dstPath string) (string, error) {
	return downloadFile(
		srcURL,
		dstPath,
		// active download
		func(fileName string, bytesTotal, bytesRead int64, progress, speed, secondsRemaining float64) {
			log.Progress(
				progress,
				"(%s / %s) %s %s: Downloading %s",
				glog.HumanReadableBytesIEC(bytesRead),
				glog.HumanReadableBytesIEC(bytesTotal),
				glog.HumanReadableRateBytesIEC(speed, "s"),
				glog.DurationShort(secondsRemaining, glog.DURATION_SCALE_AVERAGE),
				glog.Auto(fileName),
			)
		},
		// successful download
		func(fileName string, bytesTotal, bytesRead int64, progress, speed, secondsRemaining float64) {
			log.ProgressSuccess(
				progress,
				"(%s) %s: Downloaded to %s",
				glog.HumanReadableBytesIEC(bytesTotal),
				glog.HumanReadableRateBytesIEC(speed, "s"),
				glog.Auto(fileName),
			)
		},
		// failed download
		func(fileName string, bytesTotal, bytesRead int64, progress, speed, secondsRemaining float64, err error) {
			log.ProgressError(
				progress,
				"(%s) %s: Downloading %s failed: %s",
				glog.HumanReadableBytesIEC(bytesTotal),
				glog.HumanReadableRateBytesIEC(speed, "s"),
				glog.Auto(fileName),
				glog.Error(err),
			)
		},
	)
}

func main() {
	glog.LoggerConfig.ShowSubsystem = false
	glog.LoggerConfig.ShowDateTime = false
	glog.LoggerConfig.ShowRuntimeMilliseconds = false
	glog.LoggerConfig.ShowIndicator = true
	glog.LoggerConfig.ShowRuntimeSeconds = true

	if len(os.Args) != 3 {
		if len(os.Args) != 2 {
			glog.LoggerConfig.ShowRuntimeSeconds = false
			glog.LoggerConfig.ShowIndicator = false
			log.Default("Usage:    %s [source URL] <destination directory>", os.Args[0])
			log.Default("Examples: %s https://proof.ovh.net/files/100Mb.dat /tmp/", os.Args[0])
			log.Default("          %s http://speedtest.belwue.net/100M", os.Args[0])
			return
		}

		storeCursor()
		for i := 5; i > 0; i-- {
			clearLine()
			storeCursor()
			log.Warning("No destination directory given, assuming current directory. You have %s seconds to cancel.", glog.Int(i))
			restoreCursor()
			time.Sleep(1000 * time.Millisecond)
		}
		clearLine()
		os.Args = append(os.Args, ".")
	}

	_, err := downloadWithProgress(os.Args[1], os.Args[2])
	if err != nil {
		log.Error(glog.Error(err))
	}
}
