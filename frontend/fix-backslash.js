const fs = require('fs');
let content = fs.readFileSync('lib/validations.ts', 'utf8');

// From hex dump: TWO backslashes in file (\\s)
// We need ONE backslash (\s) for regex
const BS = String.fromCharCode(92);
const doubleBS = BS + BS;
const singleBS = BS;

console.log('Looking for:', '/execute[' + doubleBS + 's(]/i,');
console.log('Replacing with:', '/execute[' + singleBS + 's(]/i,');

content = content.replace(
  '/execute[' + doubleBS + 's(]/i,',
  '/execute[' + singleBS + 's(]/i,'
);

fs.writeFileSync('lib/validations.ts', content, 'utf8');
console.log('Replaced successfully');

// Verify
const verify = fs.readFileSync('lib/validations.ts', 'utf8');
const idx = verify.indexOf('/execute[');
const snippet = verify.substring(idx, idx + 17);
console.log('\nVerification:', snippet);
console.log('Backslashes:', snippet.split('').filter(c => c.charCodeAt(0) === 92).length);
