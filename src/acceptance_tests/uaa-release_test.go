package acceptance_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os/exec"
	"github.com/onsi/gomega/gexec"
	"time"
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
	. "github.com/onsi/ginkgo/extensions/table"
	"fmt"
)

type row struct {
	Stdout   string `json:"stdout"`
	ExitCode string `json:"exit_code"`
}

type table struct {
	Rows []row
}

type sshResult struct {
	Tables []table
}

var _ = Describe("UaaRelease", func() {

	AfterEach(func() {
		deleteUAA()
	})

	DescribeTable("uaa truststore", func(addedCertificates int, optFiles ...string) {
		numCertificatesBeforeDeploy := getNumOfOSCertificates()
		deployUAA(optFiles...)
		numCertificatesAfterDeploy := getNumOfOSCertificates()
		Expect(numCertificatesAfterDeploy).To(Equal(numCertificatesBeforeDeploy + addedCertificates))

		cmd := exec.Command(boshBinaryPath, []string{"ssh", "uaa", "-c", "sudo /var/vcap/packages/uaa/jdk/bin/keytool --keystore /var/vcap/data/uaa/cert-cache/cacerts --storepass changeit -list"}...)
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		output := string(session.Out.Contents())
		Expect(output).To(MatchRegexp(fmt.Sprintf("Your keystore contains %d entries", numCertificatesAfterDeploy)))
	},
		Entry("without BPM enabled", 0, "./opsfiles/disable-bpm.yml", "./opsfiles/os-conf-0-certificate.yml"),
		Entry("without BPM enabled and os-conf", 1, "./opsfiles/disable-bpm.yml", "./opsfiles/os-conf-1-certificate.yml"),
		Entry("with BPM enabled", 0, "./opsfiles/enable-bpm.yml", "./opsfiles/os-conf-0-certificate.yml"),
		Entry("with BPM enabled and os-conf", 1, "./opsfiles/enable-bpm.yml", "./opsfiles/os-conf-1-certificate.yml"),
	)
})

func getNumOfOSCertificates() int {
	caCertificatesSSHStdoutCmd := exec.Command(boshBinaryPath, []string{"--json", "ssh", "--results", "uaa", "-c", "sudo grep 'END CERTIFICATE' /etc/ssl/certs/ca-certificates.crt | wc -l"}...)
	session, err := gexec.Start(caCertificatesSSHStdoutCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 10*time.Second).Should(gexec.Exit(0))

	var result = &sshResult{}
	err = json.Unmarshal(session.Out.Contents(), result)
	Expect(err).NotTo(HaveOccurred())
	Expect(result.Tables).To(HaveLen(1))
	Expect(result.Tables[0].Rows).To(HaveLen(1))

	numOfCerts, err := strconv.Atoi(
		strings.TrimFunc(string(result.Tables[0].Rows[0].Stdout), func(r rune) bool {
			return !unicode.IsNumber(r)
		}),
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(numOfCerts).To(BeNumerically(">=", 148))
	return numOfCerts
}