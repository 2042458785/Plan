import { FileBlob, SpreadsheetFile } from "@oai/artifact-tool";

const workbookPath =
  process.env.COCO_INSPECT_PATH ||
  "/Users/ddy/GolandProjects/Plans/Business-Plan/外贸/outputs/coco-living-space-20260730/COCO_Living_Space_高销量商品与1688搜货表.xlsx";

const input = await FileBlob.load(workbookPath);
const workbook = await SpreadsheetFile.importXlsx(input);

const result = {
  sheets: (
    await workbook.inspect({
      kind: "sheet",
      include: "id,name",
      maxChars: 4000,
    })
  ).ndjson,
  overview: (
    await workbook.inspect({
      kind: "table",
      range: "概览!A1:H15",
      include: "values,formulas",
      tableMaxRows: 18,
      tableMaxCols: 10,
      maxChars: 8000,
    })
  ).ndjson,
  allHead: (
    await workbook.inspect({
      kind: "table",
      range: "全部商品!A1:Z8",
      include: "values,formulas",
      tableMaxRows: 10,
      tableMaxCols: 26,
      maxChars: 18000,
    })
  ).ndjson,
  allTail: (
    await workbook.inspect({
      kind: "table",
      range: "全部商品!A885:Z889",
      include: "values,formulas",
      tableMaxRows: 8,
      tableMaxCols: 26,
      maxChars: 14000,
    })
  ).ndjson,
  progressRows: (
    await workbook.inspect({
      kind: "table",
      range: "全部商品!A43:Z59",
      include: "values,formulas",
      tableMaxRows: 20,
      tableMaxCols: 26,
      maxChars: 30000,
    })
  ).ndjson,
  verifiedHead: (
    await workbook.inspect({
      kind: "table",
      range: "高销量深度核验!A1:Z7",
      include: "values,formulas",
      tableMaxRows: 10,
      tableMaxCols: 28,
      maxChars: 14000,
    })
  ).ndjson,
  verifiedTail: (
    await workbook.inspect({
      kind: "table",
      range: "高销量深度核验!A29:Z31",
      include: "values,formulas",
      tableMaxRows: 6,
      tableMaxCols: 28,
      maxChars: 10000,
    })
  ).ndjson,
  highHead: (
    await workbook.inspect({
      kind: "table",
      range: "高销量商品!A1:R7",
      include: "values,formulas",
      tableMaxRows: 10,
      tableMaxCols: 20,
      maxChars: 10000,
    })
  ).ndjson,
  errors: (
    await workbook.inspect({
      kind: "match",
      searchTerm: "#REF!|#DIV/0!|#VALUE!|#NAME\\?|#N/A",
      options: { useRegex: true, maxResults: 100 },
      summary: "status formula error scan",
    })
  ).ndjson,
};

console.log(JSON.stringify(result, null, 2));
