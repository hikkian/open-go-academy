package main

func explainDiagnostic(message string, stage string) string {
 switch stage { case "path": return "Проверь папку и имя файла"; case "tool": return "Проверь установку и открой новый терминал"; default: return "Прочитай первую ошибку и указанную строку" }
}
