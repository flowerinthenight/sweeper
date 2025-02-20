package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/buraksezer/consistent"
	backoffv4 "github.com/cenkalti/backoff/v4"
	"github.com/cespare/xxhash"
	ll "github.com/flowerinthenight/sweeper/log"
	gaxv2 "github.com/googleapis/gax-go/v2"
	"github.com/spf13/cobra"
)

var (
	sum = func(m map[string]float64, key string, val float64) {
		v, ok := m[key]
		if ok {
			v += val
			m[key] = v
		} else {
			m[key] = val
		}
	}
)

func testGaxBackoff() {
	bo := gaxv2.Backoff{
		Initial:    time.Second,
		Max:        time.Minute, // Maximum amount of time between retries.
		Multiplier: 2,
	}

	var cnt int
	operation := func() error {
		cnt++
		ll.Info("cnt:", cnt)
		if cnt >= 50 {
			return nil
		}

		return fmt.Errorf("backoff")
	}

	for {
		err := operation()
		if err != nil {
			time.Sleep(bo.Pause())
			continue
		}

		break
	}
}

func testBackoffv4() {
	var cnt int
	operation := func() error {
		cnt++
		ll.Info("cnt:", cnt)
		if cnt >= 100 {
			return nil
		}

		return fmt.Errorf("backoff")
	}

	err := backoffv4.Retry(operation, backoffv4.NewExponentialBackOff())
	if err != nil {
		ll.Fail("final backoff failed")
	}
}

type Member string

func (m Member) String() string {
	return string(m)
}

type hasher struct{}

func (h hasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

func testConsistent() {
	mathrand.Seed(time.Now().UTC().UnixNano())
	members := []consistent.Member{}
	for i := 0; i < 8; i++ {
		member := Member(fmt.Sprintf("node%d.olricmq", i))
		members = append(members, member)
	}
	cfg := consistent.Config{
		// PartitionCount:    71,
		PartitionCount:    7919,
		ReplicationFactor: 20,
		Load:              1.25,
		Hasher:            hasher{},
	}

	c := consistent.New(members, cfg)
	owners := make(map[string]int)
	for partID := 0; partID < cfg.PartitionCount; partID++ {
		owner := c.GetPartitionOwner(partID)
		owners[owner.String()]++
	}
	fmt.Println("average load:", c.AverageLoad())
	fmt.Println("owners:", owners)

	// --------------------------------------------------------

	// members := []consistent.Member{}
	// for i := 0; i < 8; i++ {
	// 	member := Member(fmt.Sprintf("node%d.olricmq", i))
	// 	members = append(members, member)
	// }
	// cfg := consistent.Config{
	// 	PartitionCount:    271,
	// 	ReplicationFactor: 40,
	// 	Load:              1.2,
	// 	Hasher:            hasher{},
	// }
	// c := consistent.New(members, cfg)

	// keyCount := 10000000
	// load := (c.AverageLoad() * float64(keyCount)) / float64(cfg.PartitionCount)
	// log.Printf("Maximum key count for a member should be around %f\n", math.Ceil(load))
	// distribution := make(map[string]int)
	// key := make([]byte, 4)
	// for i := 0; i < keyCount; i++ {
	// 	rand.Read(key)
	// 	member := c.LocateKey(key)
	// 	distribution[member.String()]++
	// }
	// for member, count := range distribution {
	// 	log.Printf("member: %s, key count: %d\n", member, count)
	// }

	// --------------------------------------------------------
	// members := []consistent.Member{}
	// for i := 0; i < 8; i++ {
	// 	member := Member(fmt.Sprintf("node%d.olricmq", i))
	// 	members = append(members, member)
	// }
	// // Modify PartitionCount, ReplicationFactor and Load to increase or decrease
	// // relocation ratio.
	// cfg := consistent.Config{
	// 	PartitionCount:    271,
	// 	ReplicationFactor: 20,
	// 	Load:              1.25,
	// 	Hasher:            hasher{},
	// }
	// c := consistent.New(members, cfg)

	// // Store current layout of partitions
	// owners := make(map[int]string)
	// for partID := 0; partID < cfg.PartitionCount; partID++ {
	// 	owners[partID] = c.GetPartitionOwner(partID).String()
	// }

	// // Add a new member
	// m := Member(fmt.Sprintf("node%d.olricmq", 9))
	// c.Add(m)

	// // Get the new layout and compare with the previous
	// var changed int
	// for partID, member := range owners {
	// 	owner := c.GetPartitionOwner(partID)
	// 	if member != owner.String() {
	// 		changed++
	// 		fmt.Printf("partID: %3d moved to %s from %s\n", partID, owner.String(), member)
	// 	}
	// }
	// fmt.Printf("\n%d%% of the partitions are relocated\n", (100*changed)/cfg.PartitionCount)
}

func testGptApiStream() {
	in := map[string]interface{}{
		"model":      "gpt-3.5-turbo",
		"max_tokens": 200,
		"stream":     true,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You will be role-playing as an AWS cost optimization expert, specializing in various cost-saving strategies such as Reserved Instances, instance resizing, and more. You have a deep understanding of public cloud costs and technical challenges, and you provide valuable information to customers. Specifically, your role is to analyze the customer's needs and propose tailored AWS services and solutions, like utilizing Reserved Instances, resizing instances, or implementing spot instances, to optimize costs and maximize the ROI of their cloud environment. My first request is, \"What methods are available to maximize ROI in an AWS environment?\" In the following chat, please strictly adhere to the role-play rules and constraints listed below.\n#Constraints\nThe first-person pronoun to refer to yourself is \"I.\"\nThe second-person pronoun to refer to the User is \"you\" or \"the customer.\"\nYour name is COVER-kun.\nCOVER-kun's tone is sophisticated and always puts the customer first.\n#Example of tone\nI am COVER-kun, here to answer your questions and help with any challenges related to AWS cost management and optimization strategies such as Reserved Instances and instance resizing!\n#Behavior guidelines\nPlease treat the User with courtesy.\nThink intellectually and strategically, and act with a plan.\nAim to create a fun atmosphere.\nEngage with positive energy.",
			},
			{
				"role":    "user",
				"content": "TEST",
			},
		},
	}

	b, _ := json.Marshal(in)
	client := http.Client{Timeout: 5 * time.Minute}
	url := "https://api.openai.com/v1/chat/completions"
	r, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		ll.Fail(err)
		return
	}

	token := os.Getenv("OPENAI_API_KEY")
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	resp, err := client.Do(r)
	if err != nil {
		ll.Fail(err)
		return
	}

	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			ll.Fail(err)
			break
		}

		if string(line) == "\n" {
			continue
		}

		ll.Info(string(line))
	}
}

func IsValidWord() {
	client, err := genai.NewClient(context.TODO(), "mobingi-main", "asia-northeast1")
	gemini := client.GenerativeModel("gemini-pro-vision")
	// prompt := genai.Text("Is this a valid word? Answer \"1\" if valid and \"0\" if invalid.\nVojVTCEJpIffjhs1v0BbN0MO9P9JWRYB")
	prompt := genai.Text("Is this a valid word? Answer \"1\" if valid and \"0\" if invalid.\nSageMaker")
	resp, err := gemini.GenerateContent(context.Background(), prompt)
	if err != nil {
		ll.Fail("error generating content:", err)
		return
	}

	rb, _ := json.MarshalIndent(resp, "", "  ")
	ll.Info(string(rb))
}

func IsStackTrace() {
	client, err := genai.NewClient(context.TODO(), "mobingi-main", "asia-northeast1")
	gemini := client.GenerativeModel("gemini-pro-vision")
	// prompt := genai.Text("Does this look like a stack trace of a crashed software program?\n\nhttps://console.cloud.google.com/errors/CJij8diFxNjisgE?project=mobingi-main&time=P30D&utm_source=error-reporting-notification&utm_medium=webhook&utm_content=resolved-error")
	// prompt := genai.Text("Does this look like a stack trace of a crashed software program?\n\nAn excerpt is a quoted fragment from a book, novel, poem, short story, article, speech, or other literary work that is used to give the reader a specific example from the source.")
	// prompt := genai.Text("Does this look like a stack trace of a crashed software program? Answer \"yes\" if you think it is and \"no\" if you think it isn't.\n\nBack-off restarting failed container linkbatchd in pod linkbatchd-d5867985b-2bmfr_default(d1c54444-6791-4476-a5dc-aa2bdf9f254b)")
	// prompt := genai.Text("Does this look like a stack trace of a crashed software program? Answer \"yes\" if you think it is and \"no\" if you think it isn't.\n\nnetwork is not ready: container runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: cni plugin not initialized")
	// prompt := genai.Text("Does this look like a stack trace of a crashed software program? Answer \"yes\" if you think it is and \"no\" if you think it isn't.\n\npanic: runtime error: index out of range [0] with length 0\n\ngoroutine 802 [running]:\ngithub.com/mobingilabs/ouchan/services/coverd/module/billing.(*svc).GetCustomerSubscriptionStatus(0xc000b9b648, {0x25be568?, 0xc0015edd70?}, 0x201d820?)\n\tgithub.com/mobingilabs/ouchan/services/coverd/module/billing/billing.go:134 +0x16e5\nmain.(*service).GetCustomerSubscriptionStatus(0x205275a?, {0x25be568, 0xc0015edd70}, 0xc001113ec0?)\n\tgithub.com/mobingilabs/ouchan/services/coverd/service.go:599 +0x4c\ngithub.com/alphauslabs/blue-sdk-go/cover/v1._Cover_GetCustomerSubscriptionStatus_Handler.func1({0x25be568, 0xc0015edd70}, {0x1d22b80?, 0xc0015edda0})\n\tgithub.com/alphauslabs/blue-sdk-go@v0.69.68/cover/v1/cover_grpc.pb.go:5362 +0x75\ngithub.com/mobingilabs/ouchan/pkg/blueinterceptors.(*UserData).AuthUnaryInterceptor(0xc000a140f0, {0x25be568, 0xc0015edd70}, {0x1d22b80, 0xc0015edda0}, 0xc0010f49c0, 0xc001648528)\n\tgithub.com/mobingilabs/ouchan/pkg/blueinterceptors/user.go:514 +0x903\ngoogle.golang.org/grpc.getChainUnaryHandler.func1({0x25be568, 0xc0015edd70}, {0x1d22b80, 0xc0015edda0})\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1203 +0xb2\nmain.run.UnaryServerInterceptor.func4({0x25be568, 0xc0015edd70}, {0x1d22b80, 0xc0015edda0}, 0xc0010f49c0, 0xc000ebafc0)\n\tgithub.com/grpc-ecosystem/go-grpc-middleware@v1.4.0/ratelimit/ratelimit.go:24 +0xc2\ngoogle.golang.org/grpc.NewServer.chainUnaryServerInterceptors.chainUnaryInterceptors.func1({0x25be568, 0xc0015edd70}, {0x1d22b80, 0xc0015edda0}, 0xc000fae998?, 0x1bf2320?)\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1194 +0x85\ngithub.com/alphauslabs/blue-sdk-go/cover/v1._Cover_GetCustomerSubscriptionStatus_Handler({0x20299c0?, 0xc0010f8b90}, {0x25be568, 0xc0015edd70}, 0xc001204c00, 0xc0010f4940)\n\tgithub.com/alphauslabs/blue-sdk-go@v0.69.68/cover/v1/cover_grpc.pb.go:5364 +0x135\ngoogle.golang.org/grpc.(*Server).processUnaryRPC(0xc0013d0000, {0x25be568, 0xc0015edce0}, {0x25c96e0, 0xc000ec7860}, 0xc000d50fc0, 0xc0013b8480, 0x390f220, 0x0)\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1386 +0xe23\ngoogle.golang.org/grpc.(*Server).handleStream(0xc0013d0000, {0x25c96e0, 0xc000ec7860}, 0xc000d50fc0)\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1797 +0x100c\ngoogle.golang.org/grpc.(*Server).serveStreams.func2.1()\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1027 +0x8b\ncreated by google.golang.org/grpc.(*Server).serveStreams.func2 in goroutine 219\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1038 +0x135")
	prompt := genai.Text("Does this look like a stack trace of a crashed software program? Answer \"yes\" if you think it is and \"no\" if you think it isn't.\n\npanic: runtime error: invalid memory address or nil pointer dereference\n[signal SIGSEGV: segmentation violation code=0x1 addr=0x28 pc=0x1872773]\n\ngoroutine 2316 [running]:\ngithub.com/mobingilabs/ouchan/services/coverd/module/costgroup.GetFilterFromCombinationAzure(...)\n\tgithub.com/mobingilabs/ouchan/services/coverd/module/costgroup/common.go:212\ngithub.com/mobingilabs/ouchan/services/coverd/module/costgroup.(*svc).getTopCost(0xc000088688, {0xc0019081e0, 0x11}, 0xc00004ec30, {0xc000ad6000, 0x2f5, 0x350}, 0xc0013b4540)\n\tgithub.com/mobingilabs/ouchan/services/coverd/module/costgroup/costusage.go:2365 +0x653\ngithub.com/mobingilabs/ouchan/services/coverd/module/costgroup.(*svc).GetCostUsage(0xc000088688, 0xc0013b4540, {0x25cbfe0?, 0xc001492620})\n\tgithub.com/mobingilabs/ouchan/services/coverd/module/costgroup/costusage.go:185 +0xac7\nmain.(*service).GetCostUsage(0xc00129e000?, 0x1f9fc80?, {0x25cbfe0, 0xc001492620})\n\tgithub.com/mobingilabs/ouchan/services/coverd/service.go:308 +0x86\ngithub.com/alphauslabs/blue-sdk-go/cover/v1._Cover_GetCostUsage_Handler({0x2031300?, 0xc0014091c0}, {0x25c9b18, 0xc00129e000})\n\tgithub.com/alphauslabs/blue-sdk-go@v0.69.70/cover/v1/cover_grpc.pb.go:4188 +0xd3\ngithub.com/mobingilabs/ouchan/pkg/blueinterceptors.(*UserData).AuthStreamInterceptor(0xc000a540f0, {0x2031300, 0xc0014091c0}, {0x25c9b18?, 0xc00129e000?}, 0xc001150c90, 0x210aa90)\n\tgithub.com/mobingilabs/ouchan/pkg/blueinterceptors/user.go:577 +0x6b7\ngoogle.golang.org/grpc.getChainStreamHandler.func1({0x2031300, 0xc0014091c0}, {0x25c9b18, 0xc00129e000})\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1532 +0xb2\nmain.run.StreamServerInterceptor.func6({0x2031300, 0xc0014091c0}, {0x25c9b18, 0xc00129e000}, 0xc001150c90, 0xc00109a580)\n\tgithub.com/grpc-ecosystem/go-grpc-middleware@v1.4.0/ratelimit/ratelimit.go:34 +0xb6\ngoogle.golang.org/grpc.NewServer.chainStreamServerInterceptors.chainStreamInterceptors.func2({0x2031300, 0xc0014091c0}, {0x25c9b18, 0xc00129e000}, 0x1a809a0?, 0xc0013d3040?)\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1523 +0x85\ngoogle.golang.org/grpc.(*Server).processStreamingRPC(0xc00142c600, {0x25c6148, 0xc001233050}, {0x25d12c0, 0xc001132820}, 0xc0012866c0, 0xc0014032f0, 0x3916940, 0x0)\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1687 +0x1267\ngoogle.golang.org/grpc.(*Server).handleStream(0xc00142c600, {0x25d12c0, 0xc001132820}, 0xc0012866c0)\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1801 +0xfbb\ngoogle.golang.org/grpc.(*Server).serveStreams.func2.1()\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1027 +0x8b\ncreated by google.golang.org/grpc.(*Server).serveStreams.func2 in goroutine 229\n\tgoogle.golang.org/grpc@v1.62.1/server.go:1038 +0x135")
	// prompt := genai.Text("Is this a valid word? Answer \"1\" if valid and \"0\" if invalid.\nSageMaker")
	resp, err := gemini.GenerateContent(context.Background(), prompt)
	if err != nil {
		ll.Fail("error generating content:", err)
		return
	}

	ll.Info(resp.Candidates[0].Content.Parts[0])

	rb, _ := json.MarshalIndent(resp, "", "  ")
	ll.Info(string(rb))
}

func genGraph() {
	txt := `Create a pdf with a graph showing the number of releases per date. The x-axis will have the dates while y-axis will have the team name. The team name in this data is "Ripple". Refer from the following CSV.

date,team,releases
2024-04-01,ripple,2
2024-04-02,ripple,5
2024-04-03,ripple,10
2024-04-04,ripple,11
2024-04-05,ripple,1
2024-04-06,ripple,2
2024-04-07,ripple,5
2024-04-08,ripple,8
2024-04-09,ripple,9
2024-04-10,ripple,1
`

	client, err := genai.NewClient(context.TODO(), "mobingi-main", "asia-northeast1")
	gemini := client.GenerativeModel("gemini-pro-vision")
	prompt := genai.Text(txt)
	resp, err := gemini.GenerateContent(context.Background(), prompt)
	if err != nil {
		ll.Fail("error generating content:", err)
		return
	}

	ll.Info(resp.Candidates[0].Content.Parts[0])

	rb, _ := json.MarshalIndent(resp, "", "  ")
	ll.Info(string(rb))
}

func TestCmd() *cobra.Command {
	testcmd := &cobra.Command{
		Use:   "test",
		Short: "Test anything",
		Long:  "Test anything.",
		Run: func(cmd *cobra.Command, args []string) {
			defer func(begin time.Time) {
				ll.Info("cmd took", time.Since(begin))
			}(time.Now())
		},
	}

	return testcmd
}
