const fs = require('fs');
const file = 'lib/validations.ts';
let lines = fs.readFileSync(file, 'utf8').split('\n');

// Fix line 180-184: swap script and HTML removal order
if (lines[179] && lines[179].includes('// Remove HTML tags')) {
  const temp1 = lines[179]; // HTML comment
  const temp2 = lines[180]; // let sanitized = ... HTML
  const temp3 = lines[182]; // script comment
  const temp4 = lines[183]; // sanitized = ... script

  lines[179] = temp3.replace('// Remove script tags and content', '// Remove script tags and content FIRST (before removing tags)');
  lines[180] = temp4.replace('  sanitized = ', '  let sanitized = ');
  lines[182] = temp1;
  lines[183] = temp2.replace('  let sanitized = ', '  sanitized = ');
}

// Fix line 226: change execute\(/ to execute[\s(]/
for (let i = 0; i < lines.length; i++) {
  if (lines[i].includes('/execute\\(/i,')) {
    lines[i] = lines[i].replace('/execute\\(/i,', '/execute[\\s(]/i,');
    break;
  }
}

fs.writeFileSync(file, lines.join('\n'), 'utf8');
console.log('Fixed!');
