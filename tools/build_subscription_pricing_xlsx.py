from __future__ import annotations

import html
import os
import posixpath
import zipfile
from pathlib import Path


THREAD_ID = os.environ.get("CODEX_THREAD_ID", "manual")
OUTPUT_DIR = Path("outputs") / THREAD_ID
OUTPUT_PATH = OUTPUT_DIR / "lumioapi-subscription-pricing.xlsx"

BASE_RATE = 0.35

PLANS = [
    {
        "name": "轻享版",
        "price": 79,
        "deposit": 93,
        "quota": 265,
        "day_limit": 21,
        "week_limit": 70,
        "policy": "低门槛入口，日限和周限卡紧",
    },
    {
        "name": "标准版",
        "price": 199,
        "deposit": 240,
        "quota": 685,
        "day_limit": 85,
        "week_limit": 230,
        "policy": "主推跑量，适当限制周消耗",
    },
    {
        "name": "专业版",
        "price": 399,
        "deposit": 488,
        "quota": 1394,
        "day_limit": 230,
        "week_limit": 600,
        "policy": "高频用户，放宽日限和周限",
    },
    {
        "name": "团队版",
        "price": 799,
        "deposit": 999,
        "quota": 2854,
        "day_limit": 520,
        "week_limit": 1300,
        "policy": "最优单价，高消耗团队档",
    },
]


def col_name(index: int) -> str:
    name = ""
    while index:
        index, remainder = divmod(index - 1, 26)
        name = chr(65 + remainder) + name
    return name


def cell_ref(row: int, col: int) -> str:
    return f"{col_name(col)}{row}"


def escape(value: object) -> str:
    return html.escape(str(value), quote=True)


def inline_string(row: int, col: int, value: object, style: int = 0) -> str:
    ref = cell_ref(row, col)
    return f'<c r="{ref}" t="inlineStr" s="{style}"><is><t>{escape(value)}</t></is></c>'


def number(row: int, col: int, value: float | int, style: int = 0) -> str:
    ref = cell_ref(row, col)
    return f'<c r="{ref}" s="{style}"><v>{value}</v></c>'


def formula(row: int, col: int, formula_text: str, cached: float | int, style: int = 0) -> str:
    ref = cell_ref(row, col)
    return f'<c r="{ref}" s="{style}"><f>{escape(formula_text)}</f><v>{cached}</v></c>'


def row_xml(row: int, cells: list[str], height: int | None = None) -> str:
    attrs = f' r="{row}"'
    if height:
        attrs += f' ht="{height}" customHeight="1"'
    return f"<row{attrs}>{''.join(cells)}</row>"


def sheet_xml() -> str:
    rows: list[str] = []

    rows.append(
        row_xml(
            1,
            [
                inline_string(1, 1, "LumioAPI 订阅计价方案", 1),
            ],
            height=28,
        )
    )
    rows.append(
        row_xml(
            2,
            [
                inline_string(
                    2,
                    1,
                    "按普通充值基准 1 刀 = ¥0.35 计算。实际到账金额可编辑，其余字段为公式或最终推荐限制。",
                    2,
                ),
            ],
            height=22,
        )
    )
    rows.append(row_xml(3, [inline_string(3, 1, "普通充值基准", 3), number(3, 2, BASE_RATE, 6)]))
    rows.append(row_xml(4, [inline_string(4, 1, "目标折扣区间", 3), inline_string(4, 2, "8.00 折 - 8.50 折", 4)]))
    rows.append(row_xml(5, [inline_string(5, 1, "整体平均折扣", 3), formula(5, 2, "AVERAGE(F8:F11)", 8.24, 7)]))

    headers = [
        "套餐名称",
        "可用额度",
        "日上限",
        "周上限",
        "套餐价格",
        "套餐折扣",
        "实际单价",
        "实际到账金额",
        "日上限真实金额",
        "周上限真实金额",
        "限制策略",
    ]
    rows.append(row_xml(7, [inline_string(7, col, header, 5) for col, header in enumerate(headers, start=1)], height=24))

    for offset, plan in enumerate(PLANS):
        row = 8 + offset
        discount = round(plan["price"] / plan["deposit"] * 10, 4)
        unit_price = round(plan["price"] / plan["quota"], 4)
        day_amount = round(plan["day_limit"] * BASE_RATE, 2)
        week_amount = round(plan["week_limit"] * BASE_RATE, 2)
        day_ratio = f"{plan['day_limit']}/{plan['quota']}"
        week_ratio = f"{plan['week_limit']}/{plan['quota']}"

        rows.append(
            row_xml(
                row,
                [
                    inline_string(row, 1, plan["name"], 8),
                    formula(row, 2, f"FLOOR(H{row}/$B$3,1)", plan["quota"], 9),
                    formula(row, 3, f"MAX(1,ROUND(B{row}*{day_ratio},0))", plan["day_limit"], 9),
                    formula(row, 4, f"MAX(C{row},ROUND(B{row}*{week_ratio},0))", plan["week_limit"], 9),
                    number(row, 5, plan["price"], 10),
                    formula(row, 6, f"E{row}/H{row}*10", discount, 11),
                    formula(row, 7, f"E{row}/B{row}", unit_price, 12),
                    number(row, 8, plan["deposit"], 10),
                    formula(row, 9, f"C{row}*$B$3", day_amount, 10),
                    formula(row, 10, f"D{row}*$B$3", week_amount, 10),
                    inline_string(row, 11, plan["policy"], 13),
                ],
                height=26,
            )
        )

    rows.append(row_xml(13, [inline_string(13, 1, "汇总", 14)], height=22))
    summary = [
        ("总套餐收入", "SUM(E8:E11)", 1476, 10),
        ("总实际到账金额", "SUM(H8:H11)", 1820, 10),
        ("加权整体折扣", "SUM(E8:E11)/SUM(H8:H11)*10", 8.1099, 11),
        ("总可用额度", "SUM(B8:B11)", 5198, 9),
    ]
    for offset, (label, formula_text, cached, style) in enumerate(summary):
        row = 14 + offset
        rows.append(row_xml(row, [inline_string(row, 1, label, 3), formula(row, 2, formula_text, cached, style)]))

    notes = [
        ("使用规则", "日上限和周上限只限制套餐额度；超过后继续使用普通余额。"),
        ("扣费顺序", "先扣套餐额度，再扣普通余额。"),
        ("额度口径", "可用额度向下取整，实际到账金额保留精确值。"),
    ]
    for offset, (label, value) in enumerate(notes):
        row = 19 + offset
        rows.append(row_xml(row, [inline_string(row, 1, label, 3), inline_string(row, 2, value, 2)]))

    columns = [
        (1, 1, 16),
        (2, 4, 13),
        (5, 5, 14),
        (6, 6, 13),
        (7, 7, 14),
        (8, 8, 16),
        (9, 10, 17),
        (11, 11, 34),
    ]
    cols_xml = "".join(f'<col min="{start}" max="{end}" width="{width}" customWidth="1"/>' for start, end, width in columns)

    return f"""<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheetViews>
    <sheetView workbookViewId="0">
      <pane ySplit="7" topLeftCell="A8" activePane="bottomLeft" state="frozen"/>
      <selection pane="bottomLeft" activeCell="H8" sqref="H8"/>
    </sheetView>
  </sheetViews>
  <sheetFormatPr defaultRowHeight="18"/>
  <cols>{cols_xml}</cols>
  <sheetData>{''.join(rows)}</sheetData>
  <autoFilter ref="A7:K11"/>
  <mergeCells count="2">
    <mergeCell ref="A1:K1"/>
    <mergeCell ref="A2:K2"/>
  </mergeCells>
  <pageMargins left="0.35" right="0.35" top="0.55" bottom="0.55" header="0.3" footer="0.3"/>
</worksheet>"""


def styles_xml() -> str:
    return """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <numFmts count="4">
    <numFmt numFmtId="164" formatCode="¥#,##0.00"/>
    <numFmt numFmtId="165" formatCode="0.00 &quot;折&quot;"/>
    <numFmt numFmtId="166" formatCode="¥0.000/&quot;刀&quot;"/>
    <numFmt numFmtId="167" formatCode="0 &quot;刀&quot;"/>
  </numFmts>
  <fonts count="5">
    <font><sz val="11"/><color rgb="FF172033"/><name val="Aptos"/></font>
    <font><b/><sz val="18"/><color rgb="FF1A2F5A"/><name val="Aptos"/></font>
    <font><b/><sz val="11"/><color rgb="FF506075"/><name val="Aptos"/></font>
    <font><b/><sz val="11"/><color rgb="FF1A2F5A"/><name val="Aptos"/></font>
    <font><b/><sz val="11"/><color rgb="FF0F8A5F"/><name val="Aptos"/></font>
  </fonts>
  <fills count="5">
    <fill><patternFill patternType="none"/></fill>
    <fill><patternFill patternType="gray125"/></fill>
    <fill><patternFill patternType="solid"><fgColor rgb="FFF8FBFF"/><bgColor indexed="64"/></patternFill></fill>
    <fill><patternFill patternType="solid"><fgColor rgb="FFEAF2FF"/><bgColor indexed="64"/></patternFill></fill>
    <fill><patternFill patternType="solid"><fgColor rgb="FFFFFFFF"/><bgColor indexed="64"/></patternFill></fill>
  </fills>
  <borders count="2">
    <border><left/><right/><top/><bottom/><diagonal/></border>
    <border>
      <left style="thin"><color rgb="FFD9E2EE"/></left>
      <right style="thin"><color rgb="FFD9E2EE"/></right>
      <top style="thin"><color rgb="FFD9E2EE"/></top>
      <bottom style="thin"><color rgb="FFD9E2EE"/></bottom>
      <diagonal/>
    </border>
  </borders>
  <cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
  <cellXfs count="15">
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"><alignment wrapText="1"/></xf>
    <xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="3" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="2" fillId="2" borderId="1" xfId="0"><alignment horizontal="center"/></xf>
    <xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="165" fontId="4" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="3" fillId="4" borderId="1" xfId="0"/>
    <xf numFmtId="167" fontId="0" fillId="4" borderId="1" xfId="0"><alignment horizontal="right"/></xf>
    <xf numFmtId="164" fontId="0" fillId="4" borderId="1" xfId="0"><alignment horizontal="right"/></xf>
    <xf numFmtId="165" fontId="4" fillId="4" borderId="1" xfId="0"><alignment horizontal="right"/></xf>
    <xf numFmtId="166" fontId="0" fillId="4" borderId="1" xfId="0"><alignment horizontal="right"/></xf>
    <xf numFmtId="0" fontId="0" fillId="4" borderId="1" xfId="0"><alignment wrapText="1"/></xf>
    <xf numFmtId="0" fontId="3" fillId="3" borderId="0" xfId="0"/>
  </cellXfs>
  <cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>"""


def workbook_xml() -> str:
    return """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="订阅方案" sheetId="1" r:id="rId1"/>
  </sheets>
  <calcPr calcId="191029" fullCalcOnLoad="1" forceFullCalc="1"/>
</workbook>"""


def write_xlsx(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    files = {
        "[Content_Types].xml": """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>""",
        "_rels/.rels": """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>""",
        "xl/_rels/workbook.xml.rels": """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>""",
        "xl/workbook.xml": workbook_xml(),
        "xl/worksheets/sheet1.xml": sheet_xml(),
        "xl/styles.xml": styles_xml(),
        "docProps/core.xml": """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:dcterms="http://purl.org/dc/terms/"
  xmlns:dcmitype="http://purl.org/dc/dcmitype/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>LumioAPI 订阅计价方案</dc:title>
  <dc:creator>Codex</dc:creator>
  <cp:lastModifiedBy>Codex</cp:lastModifiedBy>
  <dcterms:created xsi:type="dcterms:W3CDTF">2026-05-31T00:00:00Z</dcterms:created>
  <dcterms:modified xsi:type="dcterms:W3CDTF">2026-05-31T00:00:00Z</dcterms:modified>
</cp:coreProperties>""",
        "docProps/app.xml": """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"
  xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>Codex</Application>
  <DocSecurity>0</DocSecurity>
  <ScaleCrop>false</ScaleCrop>
  <HeadingPairs>
    <vt:vector size="2" baseType="variant">
      <vt:variant><vt:lpstr>Worksheets</vt:lpstr></vt:variant>
      <vt:variant><vt:i4>1</vt:i4></vt:variant>
    </vt:vector>
  </HeadingPairs>
  <TitlesOfParts>
    <vt:vector size="1" baseType="lpstr">
      <vt:lpstr>订阅方案</vt:lpstr>
    </vt:vector>
  </TitlesOfParts>
  <Company>LumioAPI</Company>
  <LinksUpToDate>false</LinksUpToDate>
  <SharedDoc>false</SharedDoc>
  <HyperlinksChanged>false</HyperlinksChanged>
  <AppVersion>16.0300</AppVersion>
</Properties>""",
    }

    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as xlsx:
        for name, content in files.items():
            normalized = posixpath.normpath(name)
            xlsx.writestr(normalized, content)


if __name__ == "__main__":
    write_xlsx(OUTPUT_PATH)
    print(OUTPUT_PATH)
