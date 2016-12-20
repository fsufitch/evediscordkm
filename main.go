package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

type idNameEntity struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type attacker struct {
	Character   idNameEntity `json:"character"`
	Corporation idNameEntity `json:"corporation"`
	Alliance    idNameEntity `json:"alliance"`
	DamageDone  int          `json:"damageDone"`
	FinalBlow   bool         `json:"finalBlow"`
	ShipType    idNameEntity `json:"shipType"`
	WeaponType  idNameEntity `json:"weaponType"`
}

type byDamage []attacker

func (arr byDamage) Len() int      { return len(arr) }
func (arr byDamage) Swap(x, y int) { arr[x], arr[y] = arr[y], arr[x] }
func (arr byDamage) Less(x, y int) bool {
	return arr[x].FinalBlow || (arr[x].DamageDone < arr[y].DamageDone)
}

type victim struct {
	Character   idNameEntity `json:"character"`
	Corporation idNameEntity `json:"corporation"`
	Alliance    idNameEntity `json:"alliance"`
	DamageTaken int          `json:"damageTalen"`
	ShipType    idNameEntity `json:"shipType"`
}

type killmail struct {
	AttackerCount int          `json:"attackerCount"`
	Attackers     []attacker   `json:"attackers"`
	Victim        victim       `json:"victim"`
	SolarSystem   idNameEntity `json:"solarSystem"`
}

type zkbMetadata struct {
	TotalValue float64 `json:"totalValue"`
}

type zkbPackage struct {
	Killmail killmail    `json:"killmail"`
	KillID   int         `json:"killID"`
	Metadata zkbMetadata `json:"zkb"`
}

type zkbRedisQResponse struct {
	Package zkbPackage `json:"package"`
}

type discordWebhookMessage struct {
	Content string `json:"content"`
}

func getRawKills(output chan []byte) {
	for {
		resp, err := http.Get("https://redisq.zkillboard.com/listen.php")
		if err != nil || resp.StatusCode != 200 {
			fmt.Fprintln(os.Stderr, "HTTP error!", err, resp.StatusCode, *resp)
			continue
		}
		data, _ := ioutil.ReadAll(resp.Body)
		output <- data
	}
}

func getKillPackages(output chan zkbPackage) {
	rawKills := make(chan []byte)
	go getRawKills(rawKills)
	for rawData := range rawKills {
		response := zkbRedisQResponse{}
		json.Unmarshal(rawData, &response)
		if response.Package.KillID == 0 {
			continue
		}

		// Ignore NPCs
		filteredAttackers := []attacker{}
		for _, att := range response.Package.Killmail.Attackers {
			if att.Character.ID != 0 {
				filteredAttackers = append(filteredAttackers, att)
			}
		}
		response.Package.Killmail.Attackers = filteredAttackers

		output <- response.Package
	}
}

func inputSplit(input string) []string {
	trimmed := strings.TrimSpace(input)
	split := strings.Split(trimmed, ",")
	filtered := []string{}
	for _, item := range split {
		if len(item) > 0 {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func entityIsRelevant(entity idNameEntity, tokens []string) bool {
	for _, token := range tokens {
		if token == string(entity.ID) || token == entity.Name {
			return true
		}
	}
	return false
}

func relevantAttackers(kill zkbPackage, characters []string, corporations []string, alliances []string) []attacker {
	attackers := []attacker{}
	for _, att := range kill.Killmail.Attackers {
		if entityIsRelevant(att.Character, characters) ||
			entityIsRelevant(att.Corporation, corporations) ||
			entityIsRelevant(att.Alliance, alliances) {
			attackers = append(attackers, att)
		}
	}
	sort.Sort(byDamage(attackers))
	return attackers
}

func relevantVictim(kill zkbPackage, characters []string, corporations []string, alliances []string) *victim {
	victim := kill.Killmail.Victim
	if entityIsRelevant(victim.Character, characters) ||
		entityIsRelevant(victim.Corporation, corporations) ||
		entityIsRelevant(victim.Alliance, alliances) {
		return &victim
	}
	return nil
}

func formatKillMessage(kill zkbPackage, attackers []attacker) string {
	otherAttackersCount := len(kill.Killmail.Attackers) - len(attackers)
	extraAttackersCount := len(attackers) - 3
	if extraAttackersCount < 0 {
		extraAttackersCount = 0
	}

	attackerNames := []string{"**" + attackers[0].Character.Name + "**"}
	if len(attackers) > 1 {
		attackerNames = append(attackerNames, attackers[1].Character.Name)
	}
	if len(attackers) > 2 {
		attackerNames = append(attackerNames, attackers[2].Character.Name)
	}
	if otherAttackersCount+extraAttackersCount > 0 {
		attackerNames = append(attackerNames, fmt.Sprintf("%d other(s)", otherAttackersCount+extraAttackersCount))
	}

	attackerSection := attackerNames[0] + " (solo)"
	if len(attackerNames) > 1 {
		commaAttackers := strings.Join(attackerNames[:len(attackerNames)-1], ", ")
		attackerSection = commaAttackers + " and " + attackerNames[len(attackerNames)-1]
	}

	return fmt.Sprintf(
		"%s killed **%s** (*%s*; *%s ISK*) in **%s** -- https://zkillboard.com/kill/%d",
		attackerSection,
		kill.Killmail.Victim.Character.Name,
		kill.Killmail.Victim.ShipType.Name,
		humanize.Commaf(kill.Metadata.TotalValue),
		kill.Killmail.SolarSystem.Name,
		kill.KillID,
	)
}

func formatLossMessage(kill zkbPackage, victim victim) string {
	countNPC := kill.Killmail.AttackerCount - len(kill.Killmail.Attackers)
	NPCSection := ""
	if countNPC > 0 {
		NPCSection = fmt.Sprintf("and %d NPC(s) ", countNPC)
	}
	return fmt.Sprintf(
		"**%s** was killed (*%s*; *%s* ISK) by %d attacker(s) %sin **%s** -- https://zkillboard.com/kill/%d",
		kill.Killmail.Victim.Character.Name,
		kill.Killmail.Victim.ShipType.Name,
		humanize.Commaf(kill.Metadata.TotalValue),
		len(kill.Killmail.Attackers),
		NPCSection,
		kill.Killmail.SolarSystem.Name,
		kill.KillID,
	)
}

func sendToDiscord(discordURL string, message string) {
	fmt.Println(message)
	if len(discordURL) == 0 {
		return
	}
	msg := discordWebhookMessage{message}
	data, _ := json.Marshal(msg)

	http.Post(discordURL, "application/json", bytes.NewBuffer(data))
}

func main() {
	charactersInput := flag.String("char", "", "comma-separated list of character names or IDs")
	corporationsInput := flag.String("corp", "", "comma-separated list of corporation names or IDs")
	alliancesInput := flag.String("all", "", "comma-separated list of alliance names or IDs")
	discordInput := flag.String("discord", "", "Discord webhook URL")
	flag.Parse()

	characters := inputSplit(*charactersInput)
	corporations := inputSplit(*corporationsInput)
	alliances := inputSplit(*alliancesInput)
	discordURL := *discordInput

	if len(characters)+len(corporations)+len(alliances) == 0 {
		flag.PrintDefaults()
		return
	}

	killPacks := make(chan zkbPackage)
	go getKillPackages(killPacks)

	for k := range killPacks {
		attackers := relevantAttackers(k, characters, corporations, alliances)
		victim := relevantVictim(k, characters, corporations, alliances)

		message := ""
		if len(attackers) > 0 {
			message = formatKillMessage(k, attackers)
		} else if victim != nil {
			message = formatLossMessage(k, *victim)
		}
		if len(message) > 0 {
			sendToDiscord(discordURL, message)
		}
	}
}
