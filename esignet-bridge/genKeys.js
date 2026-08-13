import { generateKeyPair, exportJWK } from 'jose';
import { writeFileSync } from 'fs';
import { randomUUID } from 'crypto';

const { publicKey, privateKey } = await generateKeyPair('RS256', {
  modulusLength: 2048,
  extractable: true,
});

const kid  = randomUUID();
const pub  = { ...(await exportJWK(publicKey)),  kid, alg: 'RS256', use: 'sig' };
const priv = { ...(await exportJWK(privateKey)), kid, alg: 'RS256', use: 'sig' };

writeFileSync('private.jwk.json', JSON.stringify(priv, null, 2));
writeFileSync('public.jwk.json',  JSON.stringify(pub,  null, 2));

console.log('Wrote private.jwk.json and public.jwk.json');
console.log('\nRegister this public key with eSignet:\n');
console.log(JSON.stringify(pub));
