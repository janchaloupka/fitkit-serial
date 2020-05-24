# FITkit Serial
Multiplatformní utilita psaná v jazyce Go pro vyhledávání, komunikaci a programování FITkit zařízení verze 1.x a 2.x.

![Go Build](https://github.com/janch32/fitkit-serial/workflows/Go%20Build/badge.svg)

[Poslední verze ke stažení](https://github.com/janch32/fitkit-serial/releases)

## Použití

### Seznam argumentů programu
* **`--help`**
    * Zobrazí nápovědu
* **`--list`**
    * Vypíše seznam detekovaných připojených FITkit zařízení. Nelze kombinovat s jinými argumenty
* **`--flash`**
    * Naprogramuje do paměti MCU a FPGA nová data
* **`--bin string`**
    * Cesta k binárnímu souboru obsahující data FPGA paměti. Použijte s `--flash`
* **`--hex1x string`**
    * Cesta k .hex souboru obsahující data paměti pro MCU v1.x MCU. Použijte s `--flash`
* **`--hex2x string`**
    * Cesta k .hex souboru obsahující data paměti pro MCU v2.x MCU. Použijte s `--flash`
* **`--force`**
    * Vynutit nahrání nových dat. Použijte s `--flash`
* **`--port string`**
    * Specifikovat, který port má být při komunikaci použit (volitelné). Pokud není specifikován žádný port, program se připojí k prvnímu detekovanému FITkit zařízení
* **`--term`**
    * Naváže spojení s FITkitem a otevře terminál. Pokud je zadáno společně s `--flash`, je terminál otevřen po úspěšném nahrání nových dat

### Nalezení připojených zařízení (`--list`)
Vypíše v JSON formátu všechny detekované FITkit zařízení připojené k počítači. Pokud nejsou detekovány žádné zařízení, vrátí program prázdné pole `[]`
#### Příklad
```
$ fitkit-serial --list

[
    {
        "Port": "COM6",
        "Version": "2.x",
        "Revision": "163"
    },
    {
        "Port": "COM10",
        "Version": "1.x",
        "Revision": "163"
    }
]
```

### Otevření terminálu (`--term`)
Naváže spojení s FITkitem v normálním režimu, kdy je spuštěn nahraný program a je možné interaktivně s programem komunikovat přes standardní vstup a výstup programu.

Zadáním volitelného argumentu `--port` se specifikuje konkrétní sériový port, ke kterému se program připojí. Pokud tento argument není zadán, program prvně provede detekci dostupných zařízení a připojí se na první nalezené zařízení.

### Příklad
```
$ fitkit-serial --term

Port not specified, running port autodiscovery...
Connecting to COM6
FITkit 2.x $Rev: 163 $

Inicializace FPGA: XC3S50
Inicializace FLASH: AT45DB041D
Programovani FPGA: ......................................... OK
Inicializace HW
>
```

```
$ fitkit-serial --term --port COM10

Connecting to COM10
FITkit 1.x $Rev: 163 $

Inicializace FPGA: XC3S50
Inicializace FLASH: AT45DB041D
Programovani FPGA: ......................................... OK
Inicializace HW
>
```

### Programování zařízení (`--flash`)
Nahraje do MCU a FPGA flash paměti FITkitu nová data. Programování je funkčně identické s utilitou [fkflash](https://merlin.fit.vutbr.cz/FITkit/docs/navody/app_fkflash.html). Při programování je nutné specifikovat .hex soubory MCUv1.x (`--hex1x`), MCUv2.x (`--hex2x`) a binární soubor FPGA dat `--bin`. Utilita využívá k rychlejší komunikaci optimalizovaný boot loader, jehož autorem je Doc. Ing. Zdeněk Vašíček, PhD.

Stejně jako u terminálu je možné zadat volitelný argument `--port` specifikující konkrétní sériový port, ke kterému se program připojí. Pokud tento argument není zadán, program prvně provede detekci dostupných zařízení a připojí se na první nalezené zařízení.

Pokud je již zadaný binární nebo .hex soubor v zařízení nahrán, je přeskočen. Opětovné nahrání lze vynutit příkazem `--force`.

Zároveň je možné také specifikovat argument `--term`. Pokud dojde k úspěšnému nahrání, tento argument způsobí, že se po nahrání otevře terminál a spustí se nahraný program.

#### Příklad
```
$ fitkit-serial --flash --hex1x build/output_f1xx.hex --hex2x build/output_f2xx.hex --bin build/output.bin

Port not specified, running port autodiscovery...
Connecting to COM6
CPU: F2x family; Device: F26F; FITkitBSLrev: 20090326
Uploading FITkit bootloader...
FITkit bootloader running
FITkit bootloader handshake...
Reading info...

Skipping MCU flash, digitest matches (identical data are already in the MCU flash)

Skipping FPGA flash, digitest matches (identical data are already in the FPGA flash)

Writing info...
Success
```

```
$ fitkit-serial --flash --hex1x build/output_f1xx.hex --hex2x build/output_f2xx.hex --bin build/output.bin --force --port COM6

Connecting to COM6
Mass erase
Transmitting default password...
CPU: F2x family; Device: F26F; FITkitBSLrev: 20090326
Uploading FITkit bootloader...
FITkit bootloader running
FITkit bootloader handshake...
Reading info...

Uploading MCU HEX data...
build/output_f2xx.hex

Uploading FPGA binary data...
build/output.bin

Writing info...
Success
```
