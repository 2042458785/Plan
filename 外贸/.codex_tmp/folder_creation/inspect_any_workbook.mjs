import { FileBlob, SpreadsheetFile } from "@oai/artifact-tool";
import path from "node:path";

const workbookPath = path.resolve(process.argv[2]);
const input = await FileBlob.load(workbookPath);
const workbook = await SpreadsheetFile.importXlsx(input);
const summary = await workbook.inspect({
  kind: "workbook,sheet,table",
  maxChars: 16000,
  tableMaxRows: 10,
  tableMaxCols: 12,
  tableMaxCellChars: 100,
});

console.log(summary.ndjson);
