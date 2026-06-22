package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	offsetsURL     = "https://raw.githubusercontent.com/a2x/cs2-dumper/main/output/offsets.json"
	clientDllURL   = "https://raw.githubusercontent.com/a2x/cs2-dumper/main/output/client_dll.json"
	offsetsFile    = "offsets.json"
	offsetsMaxAge  = 6 * time.Hour
	boneArrayDelta = 0x80
)

type Offset struct {
	DwViewMatrix           uintptr `json:"dwViewMatrix"`
	DwLocalPlayerPawn      uintptr `json:"dwLocalPlayerPawn"`
	DwEntityList           uintptr `json:"dwEntityList"`
	M_hPlayerPawn          uintptr `json:"m_hPlayerPawn"`
	M_iHealth              uintptr `json:"m_iHealth"`
	M_lifeState            uintptr `json:"m_lifeState"`
	M_iTeamNum             uintptr `json:"m_iTeamNum"`
	M_vOldOrigin           uintptr `json:"m_vOldOrigin"`
	M_pGameSceneNode       uintptr `json:"m_pGameSceneNode"`
	M_modelState           uintptr `json:"m_modelState"`
	M_boneArray            uintptr `json:"m_boneArray"`
	M_bDormant             uintptr `json:"m_bDormant"`
	M_iszPlayerName        uintptr `json:"m_iszPlayerName"`
	M_bPawnIsAlive         uintptr `json:"m_bPawnIsAlive"`
}

func (o Offset) valid() bool {
	return o.DwEntityList != 0 && o.DwLocalPlayerPawn != 0 && o.DwViewMatrix != 0
}

type schemaDump struct {
	ClientDLL struct {
		Classes map[string]struct {
			Fields map[string]uint64 `json:"fields"`
		} `json:"classes"`
	} `json:"client.dll"`
}

type clientOffsets struct {
	DwEntityList      uint64 `json:"dwEntityList"`
	DwLocalPlayerPawn uint64 `json:"dwLocalPlayerPawn"`
	DwViewMatrix      uint64 `json:"dwViewMatrix"`
}

type offsetsDump struct {
	ClientDLL clientOffsets `json:"client.dll"`
}

func schemaField(schema schemaDump, class, field string) (uintptr, error) {
	cls, ok := schema.ClientDLL.Classes[class]
	if !ok {
		return 0, fmt.Errorf("class %q not found", class)
	}
	value, ok := cls.Fields[field]
	if !ok {
		return 0, fmt.Errorf("field %q not found in %q", field, class)
	}
	return uintptr(value), nil
}

func downloadJSON(url string, dest any) error {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "rav3n/2.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

func buildOffsetsFromDumper() (Offset, error) {
	var offsetsData offsetsDump
	var schema schemaDump
	if err := downloadJSON(offsetsURL, &offsetsData); err != nil {
		return Offset{}, fmt.Errorf("offsets.json: %w", err)
	}
	if err := downloadJSON(clientDllURL, &schema); err != nil {
		return Offset{}, fmt.Errorf("client_dll.json: %w", err)
	}

	modelState, err := schemaField(schema, "CSkeletonInstance", "m_modelState")
	if err != nil {
		return Offset{}, err
	}

	m := func(class, field string) uintptr {
		v, err := schemaField(schema, class, field)
		if err != nil {
			log.Fatalf("schema field %s::%s: %v", class, field, err)
		}
		return v
	}

	offsets := Offset{
		DwEntityList:      uintptr(offsetsData.ClientDLL.DwEntityList),
		DwLocalPlayerPawn: uintptr(offsetsData.ClientDLL.DwLocalPlayerPawn),
		DwViewMatrix:      uintptr(offsetsData.ClientDLL.DwViewMatrix),
		M_hPlayerPawn:     m("CCSPlayerController", "m_hPlayerPawn"),
		M_iszPlayerName:   m("CBasePlayerController", "m_iszPlayerName"),
		M_bPawnIsAlive:    m("CCSPlayerController", "m_bPawnIsAlive"),
		M_vOldOrigin:      m("C_BasePlayerPawn", "m_vOldOrigin"),
		M_modelState:      modelState,
		M_boneArray:       modelState + boneArrayDelta,
		M_bDormant:        m("CGameSceneNode", "m_bDormant"),
		M_iHealth:         m("C_BaseEntity", "m_iHealth"),
		M_lifeState:       m("C_BaseEntity", "m_lifeState"),
		M_iTeamNum:        m("C_BaseEntity", "m_iTeamNum"),
		M_pGameSceneNode:  m("C_BaseEntity", "m_pGameSceneNode"),
	}
	return offsets, nil
}

func saveOffsets(offsets Offset) error {
	file, err := os.Create(offsetsFile)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(offsets)
}

func loadOffsetsFromFile() (Offset, error) {
	file, err := os.Open(offsetsFile)
	if err != nil {
		return Offset{}, err
	}
	defer file.Close()
	var offsets Offset
	if err := json.NewDecoder(file).Decode(&offsets); err != nil {
		return Offset{}, err
	}
	if !offsets.valid() {
		return Offset{}, fmt.Errorf("cached offsets.json is incomplete")
	}
	return offsets, nil
}

func offsetsFileFresh() bool {
	info, err := os.Stat(offsetsFile)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < offsetsMaxAge
}

func loadOffsets() Offset {
	if offsetsFileFresh() {
		if offsets, err := loadOffsetsFromFile(); err == nil {
			log.Println("Using cached offsets.json")
			return offsets
		}
	}

	if offsets, err := buildOffsetsFromDumper(); err == nil {
		if saveErr := saveOffsets(offsets); saveErr != nil {
			log.Printf("Warning: could not save offsets.json: %v", saveErr)
		} else {
			log.Println("Offsets updated from cs2-dumper")
		}
		return offsets
	} else {
		log.Printf("Warning: could not fetch offsets online: %v", err)
	}

	offsets, err := loadOffsetsFromFile()
	if err != nil {
		log.Fatalf("No valid offsets available: %v", err)
	}
	log.Println("Using stale cached offsets.json — connect to the internet to refresh")
	return offsets
}
