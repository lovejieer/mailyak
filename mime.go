package mailyak

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	rp "math/rand"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"time"
)

func (m *MailYak) buildMime() (*bytes.Buffer, error) {
	mb, err := randomBoundary()
	if err != nil {
		return nil, err
	}

	ab, err := randomBoundary()
	if err != nil {
		return nil, err
	}

	return m.buildMimeWithMultiParts(mb, ab)
}

// randomBoundary returns a random hexadecimal string used for separating MIME
// parts.
//
// The returned string must be sufficiently random to prevent malicious users
// from performing content injection attacks.
func randomBoundary() (string, error) {
	buf := make([]byte, 12)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("_----------=_%x", buf), nil
}


func (m *MailYak) buildMimeWithMultiParts(mb, ab string) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	//defer buf.Reset()

	if err := m.writeHeaders2(&buf); err != nil {
		return nil, err
	}

	if err := m.writeBody2(&buf); err != nil {
		return nil, err
	}


	return &buf, nil

}

// buildMimeWithBoundaries creates the MIME message using mb and ab as MIME
// boundaries, and returns the generated MIME data as a buffer.
func (m *MailYak) buildMimeWithBoundaries(mb, ab string) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	if err := m.writeHeaders2(&buf); err != nil {
		return nil, err
	}

	// Start our multipart/mixed part
	mixed := multipart.NewWriter(&buf)
	if err := mixed.SetBoundary(mb); err != nil {
		return nil, err
	}
	defer mixed.Close()

	fmt.Fprintf(&buf, "Content-Type: multipart/mixed;\r\n\tboundary=\"%s\"; charset=UTF-8\r\n\r\n", mixed.Boundary())

	ctype := fmt.Sprintf("multipart/alternative;\r\n\tboundary=\"%s\"", ab)

	altPart, err := mixed.CreatePart(textproto.MIMEHeader{"Content-Type": {ctype}})
	if err != nil {
		return nil, err
	}

	if err := m.writeBody(altPart, ab); err != nil {
		return nil, err
	}

	if err := m.writeAttachments(mixed, lineSplitterBuilder{}); err != nil {
		return nil, err
	}

	return &buf, nil

}

// writeHeaders writes the Mime-Version, Date, Reply-To, From, To and Subject headers,
// plus any custom headers set via AddHeader().
func (m *MailYak) writeHeaders2(buf io.Writer) error {


	fmt.Fprintf(buf, "x-sender: %s\r\n", m.xsender)

	fmt.Fprintf(buf, "x-receiver: %s\r\n", m.xreceiver)

	if _, err := buf.Write([]byte(m.fromHeader())); err != nil {
		return err
	}

	fmt.Fprintf(buf, "Date: %s\r\n", m.date)

	if m.replyTo != "" {
		fmt.Fprintf(buf, "Reply-To: %s\r\n", m.replyTo)
	}

	fmt.Fprintf(buf, "Subject: %s\r\n", m.subject)

	for _, to := range m.toAddrs {
		fmt.Fprintf(buf, "To: %s\r\n", to)
	}


	for k, v := range m.headers {
		fmt.Fprintf(buf, "%s: %s\r\n", k, v)
	}


	if _, err := buf.Write([]byte("Content-Transfer-Encoding: quoted-printable\r\n")); err != nil {
		return err
	}
	if _, err := buf.Write([]byte("Content-Type: text/html; charset=UTF-8\r\n")); err != nil {
		return err
	}
	if _, err := buf.Write([]byte("Mime-Version: 1.0\r\n\r\n")); err != nil {
		return err
	}



	return nil
}


func Shuffle2(vals []string) []string {
	r := rp.New(rp.NewSource(time.Now().UnixNano()+rp.Int63n(rp.Int63())))
	ret := make([]string, len(vals))
	perm := r.Perm(len(vals))
	for i, randIndex := range perm {
		ret[i] = vals[randIndex]
	}
	return ret
}

func (m *MailYak) writeHeaders_shaffle(buf io.Writer) error {


//	rp.Seed(time.Now().UnixNano())
//	x := rp.Intn(9999)+50000
//	x2:= fmt.Sprint(x)
	var ah []string
	var pm []string

//	received := fmt.Sprintf("Received: from [127.0.0.1] ([127.0.0.1:%s] helo=localhost) by oub.messagereach.com (envelope-from <%s>) (ecelerity 4.2.31.59853 r(Core:4.2.31.1)) with ESMTP id D2/FB-23957-46EC8FD5; %s",x2,m.xsender,m.date)
	pm = append(pm, "x-sender: "+m.xsender+"\r\n")
	pm = append(pm, "x-receiver: "+m.xreceiver+"\r\n")
//	pm = append(pm, received+"\r\n")
	pm = append(pm,"")
	ah = append(ah, m.fromHeader())
	ah = append(ah, "Date: "+m.date+ "\r\n")
	ah = append(ah, "Reply-To: "+m.replyTo+"\r\n")
	ah = append(ah, "Subject: "+m.subject+"\r\n")
	for _, to := range m.toAddrs {
		ah = append(ah, "To: "+to+"\r\n")
	}

	for k, v := range m.headers {
		ah = append(ah, k+": "+v+"\r\n")
	}
	ah = append(ah, "Content-Transfer-Encoding: quoted-printable\r\n")
	ah = append(ah, "Content-Type: text/html; charset=UTF-8\r\n")
	ah = append(ah, "Mime-Version: 1.0\r\n")
	ah = Shuffle2(ah)
	ah = append(pm, ah...)
	ah = append(ah, "\r\n")

	for _, bilibili := range ah {
		fmt.Fprintf(buf, "%s", bilibili)
	}

	return nil

}


// fromHeader returns a correctly formatted From header, optionally with a name
// component.
func (m *MailYak) fromHeader() string {
	if m.fromName == "" {
		return fmt.Sprintf("From: %s\r\n", m.fromAddr)
	}

	return fmt.Sprintf("From: %s <%s>\r\n", m.fromName, m.fromAddr)
}

// writeBody writes the text/plain and text/html mime parts.
func (m *MailYak) writeBody(w io.Writer, boundary string) error {
	alt := multipart.NewWriter(w)
	defer alt.Close()

	if err := alt.SetBoundary(boundary); err != nil {
		return err
	}

	var err error
	writePart := func(ctype string, data []byte) {
		if len(data) == 0 || err != nil {
			return
		}

		c := fmt.Sprintf("%s; charset=UTF-8", ctype)

		var part io.Writer
		part, err = alt.CreatePart(textproto.MIMEHeader{"Content-Type": {c}, "Content-Transfer-Encoding": {"quoted-printable"}})
		if err != nil {
			return
		}

		var buf bytes.Buffer
		qpw := quotedprintable.NewWriter(&buf)
		_, err = qpw.Write(data)
		qpw.Close()

		_, err = part.Write(buf.Bytes())
	}

	writePart("text/plain", m.plain.Bytes())
	writePart("text/html", m.html.Bytes())

	return err
}

func (m *MailYak) writeBody2_bak(w io.Writer, boundary string) error {
	alt := multipart.NewWriter(w)
	defer alt.Close()

	if err := alt.SetBoundary(boundary); err != nil {
		return err
	}
	fmt.Fprintf(w, "Content-Type: multipart/alternative;\r\n\tboundary=\"%s\"; charset=UTF-8\r\n\r\n", alt.Boundary())

	var err error
	writePart := func(ctype string, data []byte) {
		if len(data) == 0 || err != nil {
			return
		}

		c := fmt.Sprintf("%s; charset=UTF-8", ctype)

		var part io.Writer
		part, err = alt.CreatePart(textproto.MIMEHeader{"Content-Type": {c}, "Content-Transfer-Encoding": {"quoted-printable"}})
		if err != nil {
			return
		}

		var buf bytes.Buffer
		qpw := quotedprintable.NewWriter(&buf)
		_, err = qpw.Write(data)
		qpw.Close()

		_, err = part.Write(buf.Bytes())
	}

	writePart("text/plain", m.plain.Bytes())
	writePart("text/html", m.html.Bytes())

	return err
}

func (m *MailYak) writeBody2(w io.Writer) error {

		var buf bytes.Buffer
		qpw := quotedprintable.NewWriter(&buf)
		_, err := qpw.Write(m.html.Bytes())
		qpw.Close()

	fmt.Fprintf(w, "%s", buf.String())

	return err
}





