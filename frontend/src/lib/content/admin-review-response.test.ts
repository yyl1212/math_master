import {it,expect} from "vitest";
import {submissionViewSchema} from "./schemas";
import {submissionView,fixtureID} from "./test-fixtures";
const approved=()=>{const s=submissionView();return {...s,status:"approved",review:{id:fixtureID,submissionId:s.id,reviewerId:s.ownerId,frozenDigest:s.frozen.frozenDigest,decision:"approve",checks:{mathematics:true,explanations:true,relationships:true,sources:true,illustrations:true},independenceNote:"Administrator self-review with responsibility for this fixed content.",note:"All five requirements have been checked.",createdAt:s.createdAt}}};
it("reads the server's approved administrator self-review",()=>{expect(submissionViewSchema.safeParse(approved()).success).toBe(true)});
it("self-review still requires complete checks and matching frozen evidence",()=>{const s=approved();s.review.checks.mathematics=false;expect(submissionViewSchema.safeParse(s).success).toBe(false);s.review.checks.mathematics=true;s.review.frozenDigest="0".repeat(64);expect(submissionViewSchema.safeParse(s).success).toBe(false)});
