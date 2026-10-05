import {it,expect} from "vitest";
import {submissionViewSchema} from "./schemas";
import {componentSubmission,fixtureID} from "@/features/question/test-fixtures";
const approved=()=>{const s=componentSubmission();return {...s,status:"approved",review:{id:fixtureID,submissionId:s.id,reviewerId:s.ownerId,frozenDigest:s.frozen.frozenDigest,decision:"approve",checks:{mathematics:true,explanations:true,objectives:true,sources:true,illustrations:true,generation:true},independenceNote:"Administrator self-review with responsibility for these fixed questions.",generationNote:"No templates are present; generation does not apply.",note:"All six requirements have been checked.",createdAt:s.createdAt}}};
it("reads the server's approved administrator question self-review",()=>{expect(submissionViewSchema.safeParse(approved()).success).toBe(true)});
it("self-review still requires all checks and fixed evidence",()=>{const s=approved();s.review.checks.generation=false;expect(submissionViewSchema.safeParse(s).success).toBe(false);s.review.checks.generation=true;s.review.frozenDigest="0".repeat(64);expect(submissionViewSchema.safeParse(s).success).toBe(false)});
