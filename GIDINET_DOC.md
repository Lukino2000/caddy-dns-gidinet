# DNS API

URL: https://api.quickservicebox.com/API/Beta/DNSAPI.asmx

- recordAdd
- recordDelete
- recordGetList

**Importante**: l'interfacciamento alle API per il servizio DNS possibile esclusivamente tramite protocollo SOAP.

### Informazioni generali sui parametri

Si riportano di seguito le informazioni sui parametri da passare per gestire i record DNS.

**DomainName**: Dominio su cui si intende effettuare l'operazione.
**HostName**: Il sistema da noi utilizzato considera come HostName l'indirizzo completo incluso il dominio.
Ad esempio per un sottodominio "www" del dominio "dominio.ext", il campo HostName del record DNS sarà "www.dominio.ext".

Per questione di compatibilità, valgono le seguenti eccezioni nell'impostazione del campo HostName:
per i record del livello del dominio stesso, l'HostName può essere indicato con "@";
per i sottodomini, il sistema accetta HostName che non terminano con il dominio considerandoli in quel caso sottodomini del dominio stesso, ad esempio un HostName "www" del dominio "dominio.ext", sarà trattato dal sistema come "www.dominio.ext", questo vale anche per i sottodomini di 4° livello e seguenti.

Le eccezioni sopra riportate valgono esclusivamente per i parametri passati con i metodi recordAdd, recordUpdate, recordDelete.
I valori dei singoli HostName dei record restituiti dal metodo recordGetList saranno invece sempre indirizzi completi del dominio.

**RecordType**: Tipo di record, sono supportati i record A, AAAA, MX, CNAME, NS, TXT, SRV, CAA (solo servizio DNS premium).

**Data**: Valore del record, nel formato previsto per il valore di RecordType utilizzato.

**TTL**: TTL del record in secondi, valori permessi:
- 60 (60 secondi)
- 300 (5 minuti)
- 600 (10 minuti)
- 900 (15 minuti)
- 1800 (30 minuti)
- 2700 (45 minuti)
- 3600 (1 ora)
- 7200 (2 ore)
- 14400 (4 ore)
- 28800 (8 ore)
- 43200 (12 ore)
- 64800 (18 ore)
- 86400 (1 giorno)
- 172800 (2 giorni)

**Priority**: Priorità del record, solo per record MX, altrimenti 0.
Valori permessi da 0 (priorità massima) a 100 (priorità minima).

**Informazioni specifiche per record CAA (solo servizio DNS premium)**
La creazione di record CAA, se supportata dal servizio DNS in uso, richiede di codificare i parametri flags, tag e value del record nel campo Data:
flags tag "value"
L'unico flag attualmente previsto dalle specifiche è il flag "Issuer Critical Flag", da indicare nel caso sia richiesto dall'autorità di certificazione con il valore 128, in assenza di flag inserire come valore 0


## recordAdd
Metodo recordAdd

### Parametri
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|accountUsername|string|Username|
|accountPasswordB64|string|Password codificata in Base64|
|record|DNSRecord|Nuovo record DNS|

Dettagli ***DNSRecord***:
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|DomainName|string|Dominio|
|HostName|string|HostName completo del record DNS|
|RecordType|string|Tipo di record|
|Data|string|Valore/indirizzo del record|
|TTL|int|TTL del record|
|Priority|int|Priorità del record - Impostato solo per record MX, altrimenti 0|

Valore restituito

Struttura dati comprendente:
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|resultCode|int|Codice dell'esito - vedi sotto|
|resultSubCode|int|Informazioni aggiuntive sul risultato - vedi sotto|
|resultText|string|Descrizione dell'esito|

### esempi

SOAP 1.2
Di seguito è riportato un esempio di richiesta e risposta SOAP 1.2. I segnaposto devono essere sostituiti con i valori appropriati.

#### Richiesta:
```
POST /API/Beta/DNSAPI.asmx HTTP/1.1
Host: api.quickservicebox.com
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordAdd xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <accountUsername>string</accountUsername>
      <accountPasswordB64>string</accountPasswordB64>
      <record>
        <DomainName>string</DomainName>
        <HostName>string</HostName>
        <RecordType>string</RecordType>
        <Data>string</Data>
        <TTL>unsignedInt</TTL>
        <Priority>unsignedShort</Priority>
      </record>
    </recordAdd>
  </soap12:Body>
</soap12:Envelope>
```

#### Risposta
```
HTTP/1.1 200 OK
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordAddResponse xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <recordAddResult>
        <resultText>string</resultText>
        <resultCode>int</resultCode>
        <resultSubCode>int</resultSubCode>
      </recordAddResult>
    </recordAddResponse>
  </soap12:Body>
</soap12:Envelope>
```

## recordDelete
Metodo recordDelete

### Parametri
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|accountUsername|string|Username|
|accountPasswordB64|string|Password codificata in Base64|
|record|DNSRecord|Record DNS da cancellare|

Dettagli ***DNSRecord***:
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|DomainName|string|Dominio|
|HostName|string|HostName completo del record DNS|
|RecordType|string|Tipo di record|
|Data|string|Valore/indirizzo del record|
|TTL|int|TTL del record|
|Priority|int|Priorità del record - Impostato solo per record MX, altrimenti 0|

Valore restituito

Struttura dati comprendente:
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|resultCode|int|Codice dell'esito - vedi sotto|
|resultSubCode|int|Informazioni aggiuntive sul risultato - vedi sotto|
|resultText|string|Descrizione dell'esito|

### esempi

SOAP 1.2
Di seguito è riportato un esempio di richiesta e risposta SOAP 1.2. I segnaposto devono essere sostituiti con i valori appropriati.

#### Richiesta:
```
POST /API/Beta/DNSAPI.asmx HTTP/1.1
Host: api.quickservicebox.com
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordDelete xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <accountUsername>string</accountUsername>
      <accountPasswordB64>string</accountPasswordB64>
      <record>
        <DomainName>string</DomainName>
        <HostName>string</HostName>
        <RecordType>string</RecordType>
        <Data>string</Data>
        <TTL>unsignedInt</TTL>
        <Priority>unsignedShort</Priority>
      </record>
    </recordDelete>
  </soap12:Body>
</soap12:Envelope>
```

#### Risposta
```
HTTP/1.1 200 OK
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordDeleteResponse xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <recordDeleteResult>
        <resultText>string</resultText>
        <resultCode>int</resultCode>
        <resultSubCode>int</resultSubCode>
      </recordDeleteResult>
    </recordDeleteResponse>
  </soap12:Body>
</soap12:Envelope>
```

## recordGetList
Restituisce la lista dei record DNS del dominio

## Parametri
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|accountUsername|string|Username|
|accountPasswordB64|string|Password codificata in Base64|
|domainName|string|Dominio|


Valore restituito
Struttura dati comprendente:

|Nome parametro|Tipo|Descrizione|
|---|---|---|
|resultCode|int|Codice dell'esito - vedi sotto|
|resultSubCode|int|Informazioni aggiuntive sul risultato - vedi sotto|
|resultText|string|Descrizione dell'esito|
|resultItems|DNSRecordListItem[]|Record DNS del dominio|

Dettagli 'DNSRecordListItem':

|Nome parametro|Tipo|Descrizione|
|---|---|---|
|DomainName|string|Dominio|
|HostName|string|HostName completo del record DNS|
|RecordType|string|Tipo di record|
|Data|string|Valore/indirizzo del record|
|TTL|int|TTL del record|
|Priority|int|Priorità del record - Impostato solo per record MX, altrimenti 0|
|ReadOnly|boolean|Il record non è modificabile|
|Suspended|boolean|Il record è sospeso|
|SuspensionReason|string|Motivo della sospensione|


### esempi

SOAP 1.2
Di seguito è riportato un esempio di richiesta e risposta SOAP 1.2. I segnaposto devono essere sostituiti con i valori appropriati.

#### Richiesta:
```
POST /API/Beta/DNSAPI.asmx HTTP/1.1
Host: api.quickservicebox.com
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordGetList xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <accountUsername>string</accountUsername>
      <accountPasswordB64>string</accountPasswordB64>
      <domainName>string</domainName>
    </recordGetList>
  </soap12:Body>
</soap12:Envelope>
```

#### Risposta
```
HTTP/1.1 200 OK
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordGetListResponse xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <recordGetListResult>
        <resultText>string</resultText>
        <resultCode>int</resultCode>
        <resultSubCode>int</resultSubCode>
        <resultItems>
          <DNSRecordListItem>
            <DomainName>string</DomainName>
            <HostName>string</HostName>
            <RecordType>string</RecordType>
            <Data>string</Data>
            <TTL>int</TTL>
            <Priority>int</Priority>
            <ReadOnly>boolean</ReadOnly>
            <Suspended>boolean</Suspended>
            <SuspensionReason>string</SuspensionReason>
          </DNSRecordListItem>
          <DNSRecordListItem>
            <DomainName>string</DomainName>
            <HostName>string</HostName>
            <RecordType>string</RecordType>
            <Data>string</Data>
            <TTL>int</TTL>
            <Priority>int</Priority>
            <ReadOnly>boolean</ReadOnly>
            <Suspended>boolean</Suspended>
            <SuspensionReason>string</SuspensionReason>
          </DNSRecordListItem>
        </resultItems>
        <resultItemCount>int</resultItemCount>
      </recordGetListResult>
    </recordGetListResponse>
  </soap12:Body>
</soap12:Envelope>
```

## recordUpdate

## Parametri
|Nome parametro|Tipo|Descrizione|
|---|---|---|
|accountUsername|string|Username|
|accountPasswordB64|string|Password codificata in Base64|
|oldRecord|DNSRecord|Record DNS da modificare|
|newRecord|DNSRecord|Nuovo record DNS|

Dettagli 'DNSRecord':

|Nome parametro|Tipo|Descrizione|
|---|---|---|
|DomainName|string|Dominio|
|HostName|string|HostName completo del record DNS|
|RecordType|string|Tipo di record|
|Data|string|Valore/indirizzo del record|
|TTL|int|TTL del record|
|Priority|int|Priorità del record - Impostato solo per record MX, altrimenti 0|


Valore restituito
Struttura dati comprendente:

|Nome parametro|Tipo|Descrizione|
|---|---|---|
|resultCode|int|Codice dell'esito - vedi sotto|
|resultSubCode|int|Informazioni aggiuntive sul risultato - vedi sotto|
|resultText|string|Descrizione dell'esito|

### esempi

SOAP 1.2
Di seguito è riportato un esempio di richiesta e risposta SOAP 1.2. I segnaposto devono essere sostituiti con i valori appropriati.

#### Richiesta:
```
POST /API/Beta/DNSAPI.asmx HTTP/1.1
Host: api.quickservicebox.com
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordUpdate xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <accountUsername>string</accountUsername>
      <accountPasswordB64>string</accountPasswordB64>
      <oldRecord>
        <DomainName>string</DomainName>
        <HostName>string</HostName>
        <RecordType>string</RecordType>
        <Data>string</Data>
        <TTL>unsignedInt</TTL>
        <Priority>unsignedShort</Priority>
      </oldRecord>
      <newRecord>
        <DomainName>string</DomainName>
        <HostName>string</HostName>
        <RecordType>string</RecordType>
        <Data>string</Data>
        <TTL>unsignedInt</TTL>
        <Priority>unsignedShort</Priority>
      </newRecord>
    </recordUpdate>
  </soap12:Body>
</soap12:Envelope>
```

#### Risposta
```
HTTP/1.1 200 OK
Content-Type: application/soap+xml; charset=utf-8
Content-Length: length

<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <recordUpdateResponse xmlns="https://api.quickservicebox.com/DNS/DNSAPI">
      <recordUpdateResult>
        <resultText>string</resultText>
        <resultCode>int</resultCode>
        <resultSubCode>int</resultSubCode>
      </recordUpdateResult>
    </recordUpdateResponse>
  </soap12:Body>
</soap12:Envelope>
```


## Valori restituiti per: resultCode

valore 0: Operazione riuscita
valore 1: Autenticazione fallita
valore 2: Operazione fallita - impossibile modificare un valore in sola lettura
valore 3: Operazione fallita - parametri non validi
valore 4: Operazione fallita - errore non definito
valore 5: Operazione fallita - oggetto non trovato
valore 6: Operazione fallita - oggetto in uso

## Valori restituiti per: resultSubCode

Il valore di resultSubCode è un codice interno e serve per identificare in modo più dettagliato l'errore. In linea generale l'errore da fornire all'utente è l'insieme di resultCode + resultSubCode (es: "Operazione fallita - parametri non validi (17)")