package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/registry"
)

func loadOrSetupConfig() (*config.Config, error) {
	cfg, err := config.Load(paths.Config)
	if err == nil {
		return cfg, nil
	}

	if _, statErr := os.Stat(paths.Config); !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("skillhub 워크스페이스가 초기화되지 않았습니다. 지금 설정하시겠습니까? [Y/n]: ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer != "" && strings.ToLower(answer) != "y" {
		return nil, fmt.Errorf("config not found at %s (run 'skillhub init' to initialize)", paths.Config)
	}

	if err := paths.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("creating directories: %w", err)
	}
	fmt.Printf("워크스페이스를 생성했습니다: %s\n", paths.Home)

	cfg = config.DefaultConfig(paths.Home)

	fmt.Print("레지스트리 URL을 입력하세요 (건너뛰려면 Enter): ")
	repoURL, _ := reader.ReadString('\n')
	repoURL = strings.TrimSpace(repoURL)

	if repoURL != "" {
		source, err := registry.ParseRepoURL(repoURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "경고: URL 파싱 실패 (%v), 레지스트리 추가를 건너뜁니다.\n", err)
		} else {
			fmt.Print("토큰을 입력하세요 (불필요하면 Enter): ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			source.Token = token

			if token != "" {
				fmt.Print("사용자 이름을 입력하세요 (GitHub Enterprise용, 불필요하면 Enter): ")
				username, _ := reader.ReadString('\n')
				username = strings.TrimSpace(username)
				source.Username = username
			}

			client := registry.NewClient()
			source.Branch = client.DetectDefaultBranch(source)
			if _, err := client.FetchIndex(source); err != nil {
				fmt.Fprintf(os.Stderr, "경고: 레지스트리 인덱스에 접근할 수 없습니다 (%v)\n", err)
				fmt.Print("그래도 레지스트리를 추가하시겠습니까? [y/N]: ")
				forceAnswer, _ := reader.ReadString('\n')
				forceAnswer = strings.TrimSpace(forceAnswer)
				if strings.ToLower(forceAnswer) == "y" {
					if err := cfg.AddRegistry(source.Name, source.URL, source.Token, source.Username, source.Branch); err != nil {
						fmt.Fprintf(os.Stderr, "경고: 레지스트리 추가 실패 (%v)\n", err)
					} else {
						fmt.Printf("레지스트리 '%s'를 추가했습니다.\n", source.Name)
					}
				}
			} else {
				if err := cfg.AddRegistry(source.Name, source.URL, source.Token, source.Username, source.Branch); err != nil {
					fmt.Fprintf(os.Stderr, "경고: 레지스트리 추가 실패 (%v)\n", err)
				} else {
					fmt.Printf("레지스트리 '%s'를 추가했습니다.\n", source.Name)
				}
			}
		}
	}

	if err := cfg.Save(paths.Config); err != nil {
		return nil, fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("설정이 완료되었습니다!")
	return cfg, nil
}
