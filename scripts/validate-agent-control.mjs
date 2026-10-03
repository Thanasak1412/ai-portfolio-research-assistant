import {
  readControlPlane,
  validateControlPlane,
} from "./agent-control-lib.mjs";

let control;
try {
  control = readControlPlane();
} catch {
  console.error(
    "AI control-plane read failed: required files must be readable JSON.",
  );
  process.exit(1);
}

const errors = validateControlPlane(control);
if (errors.length > 0) {
  console.error("AI control-plane validation failed:");
  for (const error of errors) console.error(`- ${error}`);
  process.exit(1);
}

console.log("AI control-plane validation passed.");
