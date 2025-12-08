package git

type Result struct{
    Updated bool
    Details []string
}

func CheckBranches(branches []string) (*Result, error){
    return &Result{Updated:false, Details:[]string{"git branches check planned"}}, nil
}

func CheckTags(enabled bool) (*Result, error){
    return &Result{Updated:false, Details:[]string{"git tags check planned"}}, nil
}
