package manifest

import (
	"encoding/xml"
	"os"
	"regexp"
)

type Maven struct {
	JAVA_VERSION  string
	MAVEN_VERSION string
	TARGET_NAME   string
}

type Gradle struct {
	JAVA_VERSION   string
	GRADLE_VERSION string
	TARGET_NAME    string
}

type pom struct {
	XMLName xml.Name `xml:"project"`

	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Packaging  string `xml:"packaging"`

	Properties struct {
		JavaVersion  string `xml:"java.version"`
		MavenVersion string `xml:"maven.version"`
		Source       string `xml:"maven.compiler.source"`
		Target       string `xml:"maven.compiler.target"`
	} `xml:"properties"`

	Build struct {
		FinalName string `xml:"finalName"`
	} `xml:"build"`
}

func ParseJavaMaven(path string) (map[string]*string, error) {
	data, err := os.ReadFile(path + "/pom.xml")
	if err != nil {
		return nil, err
	}

	var pom pom
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, err
	}

	// 1️⃣ Java Version 解析优先级
	javaVersion := pom.Properties.JavaVersion
	if javaVersion == "" {
		javaVersion = pom.Properties.Target
	}
	if javaVersion == "" {
		javaVersion = pom.Properties.Source
	}
	if javaVersion == "" {
		javaVersion = "8"
	}

	mavenVersion := pom.Properties.MavenVersion
	if mavenVersion == "" {
		mavenVersion = "3.9.9"
	}

	// 2️⃣ 产物名解析优先级
	targetName := pom.Build.FinalName
	if targetName == "" {
		// 默认规则：artifactId-version
		if pom.Version != "" {
			targetName = pom.ArtifactId + "-" + pom.Version
		} else {
			targetName = pom.ArtifactId
		}
	}

	packaging := "jar"
	if pom.Packaging != "" {
		packaging = pom.Packaging
	}
	targetName += "." + packaging

	return map[string]*string{
		"JAVA_VERSION":  &javaVersion,
		"MAVEN_VERSION": &mavenVersion,
		"TARGET_NAME":   &targetName,
	}, nil
}

func ParseJavaGradle(page string) (Gradle, error) {
	data, err := os.ReadFile(page)
	if err != nil {
		return Gradle{}, err
	}

	content := string(data)

	javaVersion := parseGradleValue(content, "sourceCompatibility")
	if javaVersion == "" {
		javaVersion = parseGradleValue(content, "targetCompatibility")
	}
	if javaVersion == "" {
		javaVersion = parseGradleToolchain(content)
	}
	if javaVersion == "" {
		javaVersion = "8"
	}

	target := parseGradleValue(content, "rootProject.name")

	return Gradle{
		JAVA_VERSION: javaVersion,
		TARGET_NAME:  target,
	}, nil
}

func parseGradleValue(content, key string) string {
	re := regexp.MustCompile(key + `\s*=?\s*['"]?([\w\.]+)['"]?`)
	m := re.FindStringSubmatch(content)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func parseGradleToolchain(content string) string {
	re := regexp.MustCompile(`languageVersion\s*=\s*JavaLanguageVersion\.of\((\d+)\)`)
	m := re.FindStringSubmatch(content)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
