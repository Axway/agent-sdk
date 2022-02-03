package log

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Request struct {
	User      UserInfo
	UserAgent string
	Host      string
	Tls       string
}

type UserInfo struct {
	ClientName string
	ClientCode int
	ClientPass string
}

func TestObscureComplexStrings(t *testing.T) {

	user := UserInfo{ClientName: "John Doe", ClientCode: 32156, ClientPass: "GHSGD&#&BGL˜X"}
	request := Request{
		User:      user,
		UserAgent: "Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8",
		Host:      "http://example.com",
		Tls:       "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCACEGm5LIJ8ecO\n4IDNS1suL/6WJE+8rU8DOQmqwOfZvKS5lPyN2gZ5khe0r0qB7IW7d1dbJky9Q+bb\nQSHCpbIt2iOa20Yo0I4P+kw8zzJPujr421WVvkPe4kqawMse1izYLByMRFERJiBx\nNuSEWnhghKsVt4u2JYy4zyzPhym6/MHO500yffj0jowS0JYbPJ9MFAXkQXfpgLN8\nzocKdIZhzzS7UOWPqA8vLdvQBG9qmIdc1WsjZM67gDqsROkvlc954ax1Qmr8EddV\nw9d73fDFTVLwnuF3Ns29mpdDOnRjQDXScmPCWpj5X9JT4CCwjUQbMofSIQOdUgIA\nXRhuB0JVAgMBAAECggEACKcGMM4xzgRAFjxL2BPopJVvwhvQG7MmrNQU+CozQP7D\nrxsHelqqp1qdKYPTKDagzwuAptNOylelaVncezgRc5HTaCq7chSuFRxYPJ/QCZ1P\nUPQZs5X5Jj3qxsySrZHR1AYfI8eWJu+Jr70C8oLesb8lhMEzuuqMuQyfPaMnydAk\nPuqAlhGsrNM0gomposUfpuY61DJw5Hjmm81kCbMlciPM7zy3TR6SGmL0qQ8BkKT4\npgY9D4gtBFJkFh1VH+10ZEV7xMdspasPhGh3Fb2sYMydczKg3+VztNrV+zdrU1jY\nMfBb3AQsjBhVvuJb8Qtcrnmh0P/6Z3979LTc9jeVIQKBgQDXKtLcM+50hznkddfm\njRVJUxU7B2ho7Kd20OC8irI3FF6OG+LSfm1/gU3L8nFBD/UaqO+PKMS0WLV3ofJx\noyuAmfSZJLQFgDkA+zlpgR+Mg1ffuhW5TrnublyOgBziiOT1CJFCmKUYtPzNS2U0\nlUYJ0jEjPtHoX8gu9ougk/C53QKBgQCYVEV8O4aD4MQTT5+4X7IXX3uock4obNIf\ncQejIaYJZ5wOFYmOe0z8ouPCLispSav2U6p+LnWHqJeG1yTn9MUnENUzjWjNHYhs\n9KOLY2CS7vUH8gmHRJ4wRVO4EbOW5SfHa1fObZ+BD5BJdOiUoWqeHd4s4G1e5c2Z\nOfoQRKQu2QKBgHyWs1oGSAD5fDApfEZnUvgOP7DabT60KZPHBxqlRORXyxiGVSSF\nSGoYOS/qxmFiGA7D21MNzDiRVSJch8H9NWdVvige9I5q3JcQ4QGSXu5B71QAsCuI\nxmilRrrMu+0AT3MC7vmc4ZwY0HkfOw7jkJaHOySpb2oabBOldtwYTb+RAoGAdoy/\nNxwsZ945Or4xE5CGTWJmHoY3BYcLUKTqyK6bRZ54+Q0R7O1Q0R0EHE9KD+viBObA\nPUty9IzkwHAXrN31wZ18D47yDQ/66LDLxuMkebW2xOQ9PiTM58xMh2hfWAQnnS+R\nOnpeNFckd8aga2vkSgH8svhGpiA6jhFs59RD4qECgYB5zdpj+W1fxhTbgpYoKtjC\n2faw3F5913BbazfaMPtON49KNgnr1gd0is+JXOHKH3Am1MhPBSovL5tDlJT10nln\nIrzmPYGnUufnZo+m0aEHnmTFD17phDOp+rHfDiNZ/HUrNIGqIljb4791xgT4Mer7\nIhILoELZvEfEQJK+qgKiMQ==\n-----END PRIVATE KEY-----",
	}

	fmt.Println(fmt.Sprintf("%s", ObscureArguments([]string{"ClientPass", "Tls"}, request)))

	expected := "[{\"User\":{\"ClientName\":\"John Doe\",\"ClientCode\":32156,\"ClientPass\": \"[redacted]\"},\"UserAgent\":\"Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8\",\"Host\":\"http://example.com\",\"Tls\": \"[redacted]\"}]"
	assert.Equal(t, expected, fmt.Sprintf("%s", ObscureArguments([]string{"ClientPass", "Tls"}, request)))
}

func TestObscureNumbers(t *testing.T) {

	user := UserInfo{ClientName: "John Doe", ClientCode: 32156, ClientPass: "GHSGD&#&BGL˜X"}
	request := Request{
		User:      user,
		UserAgent: "Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8",
		Host:      "http://example.com",
		Tls:       "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCACEGm5LIJ8ecO\n4IDNS1suL/6WJE+8rU8DOQmqwOfZvKS5lPyN2gZ5khe0r0qB7IW7d1dbJky9Q+bb\nQSHCpbIt2iOa20Yo0I4P+kw8zzJPujr421WVvkPe4kqawMse1izYLByMRFERJiBx\nNuSEWnhghKsVt4u2JYy4zyzPhym6/MHO500yffj0jowS0JYbPJ9MFAXkQXfpgLN8\nzocKdIZhzzS7UOWPqA8vLdvQBG9qmIdc1WsjZM67gDqsROkvlc954ax1Qmr8EddV\nw9d73fDFTVLwnuF3Ns29mpdDOnRjQDXScmPCWpj5X9JT4CCwjUQbMofSIQOdUgIA\nXRhuB0JVAgMBAAECggEACKcGMM4xzgRAFjxL2BPopJVvwhvQG7MmrNQU+CozQP7D\nrxsHelqqp1qdKYPTKDagzwuAptNOylelaVncezgRc5HTaCq7chSuFRxYPJ/QCZ1P\nUPQZs5X5Jj3qxsySrZHR1AYfI8eWJu+Jr70C8oLesb8lhMEzuuqMuQyfPaMnydAk\nPuqAlhGsrNM0gomposUfpuY61DJw5Hjmm81kCbMlciPM7zy3TR6SGmL0qQ8BkKT4\npgY9D4gtBFJkFh1VH+10ZEV7xMdspasPhGh3Fb2sYMydczKg3+VztNrV+zdrU1jY\nMfBb3AQsjBhVvuJb8Qtcrnmh0P/6Z3979LTc9jeVIQKBgQDXKtLcM+50hznkddfm\njRVJUxU7B2ho7Kd20OC8irI3FF6OG+LSfm1/gU3L8nFBD/UaqO+PKMS0WLV3ofJx\noyuAmfSZJLQFgDkA+zlpgR+Mg1ffuhW5TrnublyOgBziiOT1CJFCmKUYtPzNS2U0\nlUYJ0jEjPtHoX8gu9ougk/C53QKBgQCYVEV8O4aD4MQTT5+4X7IXX3uock4obNIf\ncQejIaYJZ5wOFYmOe0z8ouPCLispSav2U6p+LnWHqJeG1yTn9MUnENUzjWjNHYhs\n9KOLY2CS7vUH8gmHRJ4wRVO4EbOW5SfHa1fObZ+BD5BJdOiUoWqeHd4s4G1e5c2Z\nOfoQRKQu2QKBgHyWs1oGSAD5fDApfEZnUvgOP7DabT60KZPHBxqlRORXyxiGVSSF\nSGoYOS/qxmFiGA7D21MNzDiRVSJch8H9NWdVvige9I5q3JcQ4QGSXu5B71QAsCuI\nxmilRrrMu+0AT3MC7vmc4ZwY0HkfOw7jkJaHOySpb2oabBOldtwYTb+RAoGAdoy/\nNxwsZ945Or4xE5CGTWJmHoY3BYcLUKTqyK6bRZ54+Q0R7O1Q0R0EHE9KD+viBObA\nPUty9IzkwHAXrN31wZ18D47yDQ/66LDLxuMkebW2xOQ9PiTM58xMh2hfWAQnnS+R\nOnpeNFckd8aga2vkSgH8svhGpiA6jhFs59RD4qECgYB5zdpj+W1fxhTbgpYoKtjC\n2faw3F5913BbazfaMPtON49KNgnr1gd0is+JXOHKH3Am1MhPBSovL5tDlJT10nln\nIrzmPYGnUufnZo+m0aEHnmTFD17phDOp+rHfDiNZ/HUrNIGqIljb4791xgT4Mer7\nIhILoELZvEfEQJK+qgKiMQ==\n-----END PRIVATE KEY-----",
	}

	fmt.Println(fmt.Sprintf("%s", ObscureArguments([]string{"ClientCode"}, request)))

	expected := "[{\"User\":{\"ClientName\":\"John Doe\",\"ClientCode\": \"[redacted]\",\"ClientPass\":\"GHSGD\\u0026#\\u0026BGL˜X\"},\"UserAgent\":\"Version 17.2.0.12 - Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.8 (KHTML, like Gecko) Beamrise/17.2.0.12 Chrome/17.0.939.0 Safari/535.8\",\"Host\":\"http://example.com\",\"Tls\":\"-----BEGIN PRIVATE KEY-----\\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCACEGm5LIJ8ecO\\n4IDNS1suL/6WJE+8rU8DOQmqwOfZvKS5lPyN2gZ5khe0r0qB7IW7d1dbJky9Q+bb\\nQSHCpbIt2iOa20Yo0I4P+kw8zzJPujr421WVvkPe4kqawMse1izYLByMRFERJiBx\\nNuSEWnhghKsVt4u2JYy4zyzPhym6/MHO500yffj0jowS0JYbPJ9MFAXkQXfpgLN8\\nzocKdIZhzzS7UOWPqA8vLdvQBG9qmIdc1WsjZM67gDqsROkvlc954ax1Qmr8EddV\\nw9d73fDFTVLwnuF3Ns29mpdDOnRjQDXScmPCWpj5X9JT4CCwjUQbMofSIQOdUgIA\\nXRhuB0JVAgMBAAECggEACKcGMM4xzgRAFjxL2BPopJVvwhvQG7MmrNQU+CozQP7D\\nrxsHelqqp1qdKYPTKDagzwuAptNOylelaVncezgRc5HTaCq7chSuFRxYPJ/QCZ1P\\nUPQZs5X5Jj3qxsySrZHR1AYfI8eWJu+Jr70C8oLesb8lhMEzuuqMuQyfPaMnydAk\\nPuqAlhGsrNM0gomposUfpuY61DJw5Hjmm81kCbMlciPM7zy3TR6SGmL0qQ8BkKT4\\npgY9D4gtBFJkFh1VH+10ZEV7xMdspasPhGh3Fb2sYMydczKg3+VztNrV+zdrU1jY\\nMfBb3AQsjBhVvuJb8Qtcrnmh0P/6Z3979LTc9jeVIQKBgQDXKtLcM+50hznkddfm\\njRVJUxU7B2ho7Kd20OC8irI3FF6OG+LSfm1/gU3L8nFBD/UaqO+PKMS0WLV3ofJx\\noyuAmfSZJLQFgDkA+zlpgR+Mg1ffuhW5TrnublyOgBziiOT1CJFCmKUYtPzNS2U0\\nlUYJ0jEjPtHoX8gu9ougk/C53QKBgQCYVEV8O4aD4MQTT5+4X7IXX3uock4obNIf\\ncQejIaYJZ5wOFYmOe0z8ouPCLispSav2U6p+LnWHqJeG1yTn9MUnENUzjWjNHYhs\\n9KOLY2CS7vUH8gmHRJ4wRVO4EbOW5SfHa1fObZ+BD5BJdOiUoWqeHd4s4G1e5c2Z\\nOfoQRKQu2QKBgHyWs1oGSAD5fDApfEZnUvgOP7DabT60KZPHBxqlRORXyxiGVSSF\\nSGoYOS/qxmFiGA7D21MNzDiRVSJch8H9NWdVvige9I5q3JcQ4QGSXu5B71QAsCuI\\nxmilRrrMu+0AT3MC7vmc4ZwY0HkfOw7jkJaHOySpb2oabBOldtwYTb+RAoGAdoy/\\nNxwsZ945Or4xE5CGTWJmHoY3BYcLUKTqyK6bRZ54+Q0R7O1Q0R0EHE9KD+viBObA\\nPUty9IzkwHAXrN31wZ18D47yDQ/66LDLxuMkebW2xOQ9PiTM58xMh2hfWAQnnS+R\\nOnpeNFckd8aga2vkSgH8svhGpiA6jhFs59RD4qECgYB5zdpj+W1fxhTbgpYoKtjC\\n2faw3F5913BbazfaMPtON49KNgnr1gd0is+JXOHKH3Am1MhPBSovL5tDlJT10nln\\nIrzmPYGnUufnZo+m0aEHnmTFD17phDOp+rHfDiNZ/HUrNIGqIljb4791xgT4Mer7\\nIhILoELZvEfEQJK+qgKiMQ==\\n-----END PRIVATE KEY-----\"}]"
	assert.Equal(t, expected, fmt.Sprintf("%s", ObscureArguments([]string{"ClientCode"}, request)))
}
