const fs = require('fs');
const path = require('path');
const { exec, execSync } = require('child_process');

const apiserverDir = path.join(__dirname, '..', 'pkg/apic/apiserver');
const project = 'git.ecd.axway.int/apigov/apic_agents_sdk';

const files = fs.readdirSync(apiserverDir);

for (file of files) {
  const dirToMake = file.split('_');
  const [, group, kind] = dirToMake;

  if (dirToMake.length > 2) {
    const modelsDir = `${apiserverDir}/models/${group}/${kind}`;
    execSync(`mkdir -p ${modelsDir}`);
    const currentFile = `${apiserverDir}/${file}`;
    const fileContent = fs.readFileSync(currentFile, { encoding: 'utf-8' });
    const newContent = fileContent
      .replace(
        'package v1',
        `package ${kind}\n import v1 "${project}/pkg/apic/apiserver/models/api/v1"`
      )
      .replace(/ ApiV1/, ' v1.ApiV1');
    fs.writeFile(currentFile, newContent, { encoding: 'utf-8' }, (err) => {
      if (err) console.error('ERROR: ', err);
      exec(`mv ${currentFile} ${modelsDir}`);
    });
  }
}
