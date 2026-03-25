package onboarding

import (
	"encoding/json"
	"testing"
)

func TestCollectedData_RequiredFieldsCovered(t *testing.T) {
	cd := &CollectedData{}
	if cd.RequiredFieldsCovered() {
		t.Error("empty data should not be covered")
	}

	cd = &CollectedData{
		Industry: "SaaS", CompanyStage: "startup", BusinessModel: "B2B",
		TeamSize: 10, OrgStructure: "flat", CurrentProjects: "API platform",
		PainPoints: []string{"hiring"}, CommTools: []string{"telegram"},
	}
	if !cd.RequiredFieldsCovered() {
		t.Error("all required fields should be covered")
	}
}

func TestProposedPlan_Validate(t *testing.T) {
	plan := validTestPlan()
	if err := plan.Validate(); err != nil {
		t.Errorf("valid plan should pass: %v", err)
	}

	bad := validTestPlan()
	bad.Mentor.PrimaryID = ""
	if err := bad.Validate(); err == nil {
		t.Error("missing mentor should fail")
	}
}

func TestProposedPlan_JSONRoundtrip(t *testing.T) {
	plan := validTestPlan()
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ProposedPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Mentor.PrimaryID != plan.Mentor.PrimaryID {
		t.Error("roundtrip mismatch")
	}
}

func validTestPlan() ProposedPlan {
	return ProposedPlan{
		Mentor: MentorPlan{PrimaryID: "musk", Reasoning: "test"},
		Board:  []SeatPlan{{SeatType: "ceo", PersonaID: "musk", Reasoning: "test"}},
		OrgDesign: OrgDesignPlan{
			Units: []OrgUnitPlan{{RefID: "eng", Name: "Engineering", UnitType: "department"}},
		},
		Policies: PolicyPlan{Framework: "okr", CheckinQuestions: []string{"q1"}},
		Schedule: SchedulePlan{Checkin: "0 9 * * 1-5", Timezone: "Asia/Manila"},
	}
}
