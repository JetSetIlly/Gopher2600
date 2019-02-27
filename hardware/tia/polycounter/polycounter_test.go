package polycounter_test

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
	"strings"
	"testing"
)

const correctSequence = "0@00@10@20@31@01@11@21@32@02@12@22@33@03@13@23@34@04@14@24@35@05@15@25@36@06@16@26@37@07@17@27@38@08@18@28@39@09@19@29@310@010@110@210@311@011@111@211@312@012@112@212@313@013@113@213@314@014@114@214@315@015@115@215@316@016@116@216@317@017@117@217@318@018@118@218@319@019@119@219@320@020@120@220@321@021@121@221@322@022@122@222@323@023@123@223@324@024@124@224@325@025@125@225@326@026@126@226@327@027@127@227@328@028@128@228@329@029@129@229@330@030@130@230@331@031@131@231@332@032@132@232@333@033@133@233@334@034@134@234@335@035@135@235@336@036@136@236@337@037@137@237@338@038@138@238@339@039@139@239@340@040@140@240@341@041@141@241@342@042@142@242@343@043@143@243@344@044@144@244@345@045@145@245@346@046@146@246@347@047@147@247@348@048@148@248@349@049@149@249@350@050@150@250@351@051@151@251@352@052@152@252@353@053@153@253@354@054@154@254@355@055@155@255@356@056@156@256@357@057@157@257@358@058@158@258@359@059@159@259@360@060@160@260@361@061@161@261@362@062@162@262@363@063@163@263@3"

func TestSequence(t *testing.T) {
	pk := polycounter.New6Bit()
	pk.SetResetPattern("111111")

	s := strings.Builder{}
	s.WriteString(pk.MachineInfoTerse())
	for pk.Tick() == false {
		s.WriteString(pk.MachineInfoTerse())
	}
	if s.String() != correctSequence {
		t.Fatalf("polycounter sequence has failed")
	}
}

func TestSync(t *testing.T) {
	pk := polycounter.New6Bit()
	pko := polycounter.New6Bit()
	pk.SetResetPoint(10)
	pko.SetResetPoint(10)
	pk.Sync(pko, 0)
	if pk.Count != 0 || pk.Phase != 0 {
		t.Fatalf("test sync failed")
	}
	pk.Sync(pko, -1)
	if pk.Count != 0 || pk.Phase != 1 {
		t.Fatalf("test sync failed (offset -1)")
	}
	pk.Sync(pko, -2)
	if pk.Count != 0 || pk.Phase != 2 {
		t.Fatalf("test sync failed (offset -2)")
	}
	pk.Sync(pko, -3)
	if pk.Count != 0 || pk.Phase != 3 {
		t.Fatalf("test sync failed (offset -3)")
	}
	pk.Sync(pko, -4)
	if pk.Count != 1 || pk.Phase != 0 {
		t.Fatalf("test sync failed (offset -4)")
	}
	pk.Sync(pko, 1)
	if pk.Count != 10 || pk.Phase != 3 {
		fmt.Printf("%d, %d", pk.Count, pk.Phase)
		t.Fatalf("test sync failed (offset +1)")
	}
	pk.Sync(pko, 2)
	if pk.Count != 10 || pk.Phase != 2 {
		fmt.Printf("%d, %d", pk.Count, pk.Phase)
		t.Fatalf("test sync failed (offset +2)")
	}
	pk.Sync(pko, 3)
	if pk.Count != 10 || pk.Phase != 1 {
		fmt.Printf("%d, %d", pk.Count, pk.Phase)
		t.Fatalf("test sync failed (offset +3)")
	}
	pk.Sync(pko, 5)
	if pk.Count != 9 || pk.Phase != 3 {
		fmt.Printf("%d, %d", pk.Count, pk.Phase)
		t.Fatalf("test sync failed (offset +5)")
	}
}
