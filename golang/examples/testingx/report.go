package testingx

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// WriteReport печатает задачи таблицей. Такой вывод удобно проверять golden-файлом.
func WriteReport(w io.Writer, tasks []Task) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ЗАДАЧА\tВЛАДЕЛЕЦ\tСРОК")
	for _, t := range tasks {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", t.Title, t.Owner, t.Due.Format("02.01.2006"))
	}
	return tw.Flush()
}
