package docker

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"

    ggcr "github.com/google/go-containerregistry/pkg/v1/remote"
    "github.com/google/go-containerregistry/pkg/name"
    "github.com/google/go-containerregistry/pkg/authn"
)

type Result struct {
    Updated bool
    Details []string
}

func Check(images []string) (*Result, error) {
    res := &Result{}
    if len(images) == 0 {
        return res, nil
    }
    for _, img := range images {
        remoteDigest, err := getRemoteDigest(img)
        if err != nil {
            return nil, err
        }
        localDigest, err := getLocalDigest(img)
        if err != nil {
            // local missing is treated as update
            res.Updated = true
            res.Details = append(res.Details, fmt.Sprintf("%s local missing; remote %s", img, remoteDigest))
            continue
        }
        if localDigest != remoteDigest {
            res.Updated = true
            res.Details = append(res.Details, fmt.Sprintf("%s digest changed local %s -> remote %s", img, localDigest, remoteDigest))
        }
    }
    return res, nil
}

func getRemoteDigest(ref string) (string, error) {
    r, err := name.ParseReference(ref)
    if err != nil {
        return "", err
    }
    // retry simple
    var last error
    for i := 0; i < 3; i++ {
        desc, err := ggcr.Head(r, ggcr.WithAuthFromKeychain(authn.DefaultKeychain))
        if err == nil {
            d := desc.Digest.String()
            if d != "" {
                return d, nil
            }
        }
        last = err
        time.Sleep(time.Duration(1<<i) * 200 * time.Millisecond)
    }
    if last != nil {
        return "", last
    }
    return "", fmt.Errorf("remote digest not found for %s", ref)
}

type dockerInspect struct {
    RepoDigests []string `json:"RepoDigests"`
    ID          string   `json:"Id"`
}

func getLocalDigest(ref string) (string, error) {
    // use docker CLI to avoid heavy client deps
    cmd := exec.Command("bash", "-lc", fmt.Sprintf("docker image inspect %q", ref))
    out, err := cmd.Output()
    if err != nil {
        return "", err
    }
    var objs []dockerInspect
    if err := json.Unmarshal(out, &objs); err != nil {
        return "", err
    }
    if len(objs) == 0 {
        return "", fmt.Errorf("no local image for %s", ref)
    }
    img := objs[0]
    if len(img.RepoDigests) > 0 {
        for _, d := range img.RepoDigests {
            if strings.HasPrefix(d, ref+"@") {
                return strings.TrimPrefix(d, ref+"@"), nil
            }
        }
        parts := strings.Split(img.RepoDigests[0], "@")
        if len(parts) == 2 {
            return parts[1], nil
        }
    }
    if img.ID != "" {
        return img.ID, nil
    }
    return "", fmt.Errorf("no local digest for %s", ref)
}
