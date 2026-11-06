package testingx

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// go test ./testingx -update перезаписывает эталон; diff эталона смотрят на ревью.
var update = flag.Bool("update", false, "обновить golden-файлы")

func TestWriteReportGolden(t *testing.T) {
	day := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	tasks := []Task{
		{Title: "отчёт за квартал", Owner: "аня", Due: day},
		{Title: "релиз", Owner: "боря", Due: day.AddDate(0, 0, 7)},
	}
	var buf bytes.Buffer
	if err := WriteReport(&buf, tasks); err != nil {
		t.Fatal(err)
	}

	golden := filepath.Join("testdata", "report.golden") // testdata игнорируется сборкой
	if *update {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("отчёт не совпал с эталоном.\ngot:\n%s\nwant:\n%s", buf.Bytes(), want)
	}
}
