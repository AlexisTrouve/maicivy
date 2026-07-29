$filePath = "lib/validations.ts"
$content = Get-Content -Path $filePath -Raw

# Fix 1: Swap the order - remove script tags BEFORE removing HTML tags
$pattern1 = '(?s)(export function sanitizeString\(input: string\): string \{[\r\n]+)  \/\/ Remove HTML tags[\r\n]+  let sanitized = input\.replace\(/<\[^>\]\*>\/g, ''''\);[\r\n]+[\r\n]+  \/\/ Remove script tags and content[\r\n]+  sanitized = sanitized\.replace\(/<script\[\\s\\S\]\*\?<\\\/script>\/gi, ''''\);'

$replacement1 = '$1  // Remove script tags and content FIRST (before removing tags)
  let sanitized = input.replace(/<script[\\s\\S]*?<\\/script>/gi, '''');

  // Remove HTML tags
  sanitized = sanitized.replace(/<[^>]*>/g, '''');'

$content = $content -replace $pattern1, $replacement1

# Fix 2: Change execute\(/i to execute[\s(]/i
$content = $content -replace '/execute\\\(/i,', '/execute[\\s(]/i,'

Set-Content -Path $filePath -Value $content -NoNewline
Write-Host "Fixed validations.ts"
