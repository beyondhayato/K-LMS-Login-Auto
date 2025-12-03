package ics

import (
	"fmt"
	"strings"
	"time"
	"klms-go/internal/ocr"
	"klms-go/internal/storage"
)

func GenerateICS(assignments []ocr.Assignment) string {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\n")
	sb.WriteString("VERSION:2.0\n")
	sb.WriteString("PRODID:-//K-LMS Auto//JP\n")
	sb.WriteString("CALSCALE:GREGORIAN\n")
	sb.WriteString("METHOD:PUBLISH\n")

	for _, task := range assignments {
		deadline, err := time.Parse("2006-01-02 15:04", task.Deadline)
		if err != nil {
			continue
		}
		start := deadline.Add(-1 * time.Hour)

		dtStart := start.Format("20060102T150405")
		dtEnd := deadline.Format("20060102T150405")
		now := time.Now().Format("20060102T150405")

		uid := storage.GenerateID(task.Course, task.Title, task.Deadline)

		sb.WriteString("BEGIN:VEVENT\n")
		sb.WriteString(fmt.Sprintf("UID:%s@klms-auto\n", uid))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\n", now))
		sb.WriteString(fmt.Sprintf("DTSTART;TZID=Asia/Tokyo:%s\n", dtStart))
		sb.WriteString(fmt.Sprintf("DTEND;TZID=Asia/Tokyo:%s\n", dtEnd))
		
		// ★変更点: タイトルを「科目名: 課題名」に変更
		// これでカレンダー登録時の名称が変わります
		sb.WriteString(fmt.Sprintf("SUMMARY:%s: %s\n", task.Course, task.Title))
		
		sb.WriteString(fmt.Sprintf("DESCRIPTION:【課題】%s\\n【科目】%s\\n【期限】%s\\n\\nK-LMS自動検知\n", task.Title, task.Course, task.Deadline))
		sb.WriteString("END:VEVENT\n")
	}

	sb.WriteString("END:VCALENDAR\n")
	return sb.String()
}