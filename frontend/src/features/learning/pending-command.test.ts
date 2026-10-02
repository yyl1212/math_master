import {LearningTestAccount} from "@/lib/learning/test-fixtures";
import {it,expect,vi} from "vitest";import {createPendingLearningCommand,pendingForActor} from "./pending-command";import {fixtureID,otherID} from "@/lib/learning/test-fixtures";
it("pending command freezes exact input, route and fresh UUID while preserving raw spelling",()=>{const input={kind:"numeric",raw:" 2 / 3 "};const command=createPendingLearningCommand({kind:"answerPractice",id:fixtureID},input,fixtureID);input.raw="5";expect(command.input).toEqual({kind:"numeric",raw:" 2 / 3 "});expect(Object.isFrozen(command.input)).toBe(true);expect(Object.isFrozen(command.route)).toBe(true);expect(command.key).toMatch(/^[0-9a-f-]{36}$/);const edited=createPendingLearningCommand(command.route,input,fixtureID);expect(edited.key).not.toBe(command.key);expect(pendingForActor(command,fixtureID)).toBe(command);expect(pendingForActor(command,otherID)).toBeNull();expect(pendingForActor(command,null)).toBeNull()});

import {renderHook,act,waitFor} from "@testing-library/react";
import {useLearningCommand} from "./pending-command";
import {getAuthContext} from "@/lib/auth/client";
import {requestLearning} from "@/lib/learning/client";
import {learningFailure} from "@/lib/learning/schemas";
vi.mock("@/lib/auth/client",()=>({getAuthContext:vi.fn(async()=>({ok:true,data:{user:{id:"11111111-1111-4111-8111-111111111111",username:"learner",roles:["learner"],mustChangePassword:false},csrfToken:"A".repeat(43)}}))}));
vi.mock("@/lib/learning/client",()=>({requestLearning:vi.fn(),bindLearningInput:vi.fn()}));
it("manual retry preserves the original input and key, account switch clears pending",async()=>{vi.mocked(requestLearning).mockResolvedValue(learningFailure());const hook=renderHook(()=>useLearningCommand(vi.fn()),{wrapper:LearningTestAccount});await act(async()=>hook.result.current.run({kind:"answerPractice",id:fixtureID},{kind:"numeric",raw:" 2 / 3 "}));expect(requestLearning).toHaveBeenCalledTimes(1);const first=vi.mocked(requestLearning).mock.calls[0];await act(async()=>hook.result.current.retry());expect(requestLearning).toHaveBeenCalledTimes(2);expect(vi.mocked(requestLearning).mock.calls[1].slice(0,3)).toEqual(first.slice(0,3));act(()=>window.dispatchEvent(new Event("math-master:auth-change")));expect(hook.result.current.pending).toBeNull();await act(async()=>hook.result.current.retry());expect(requestLearning).toHaveBeenCalledTimes(2)});
it("rapid repeated clicks create only one in-flight command",async()=>{vi.mocked(requestLearning).mockResolvedValue(learningFailure());const hook=renderHook(()=>useLearningCommand(vi.fn()),{wrapper:LearningTestAccount});await act(async()=>{await Promise.all([hook.result.current.run({kind:"abandonPractice",id:fixtureID},{}),hook.result.current.run({kind:"abandonPractice",id:fixtureID},{})])});expect(requestLearning).toHaveBeenCalledTimes(1)});
it("local numeric limit reports the format error without sending or losing caller input",async()=>{const hook=renderHook(()=>useLearningCommand(vi.fn()),{wrapper:LearningTestAccount});await act(async()=>hook.result.current.run({kind:"answerPractice",id:fixtureID},{kind:"numeric",raw:"1".repeat(129)}));expect(hook.result.current.error?.code).toBe("ANSWER_FORMAT_INVALID");expect(requestLearning).not.toHaveBeenCalled()});

it("the ten-second command deadline includes identity preparation and keeps an unconfirmed request for manual retry",async()=>{
 const context=await getAuthContext();vi.useFakeTimers();
 vi.mocked(getAuthContext).mockImplementationOnce(()=>new Promise(resolve=>setTimeout(()=>resolve(context),4000)));
 vi.mocked(requestLearning).mockImplementation((_route,_input,_key,signal)=>new Promise(resolve=>signal!.addEventListener("abort",()=>resolve(learningFailure()),{once:true})));
 const hook=renderHook(()=>useLearningCommand(vi.fn()),{wrapper:LearningTestAccount});
 try {
  act(()=>{void hook.result.current.run({kind:"abandonPractice",id:fixtureID},{})});
  await act(async()=>{await vi.advanceTimersByTimeAsync(4000)});
  expect(requestLearning).toHaveBeenCalledTimes(1);expect(hook.result.current.busy).toBe(true);
  await act(async()=>{await vi.advanceTimersByTimeAsync(5999)});expect(hook.result.current.busy).toBe(true);
  await act(async()=>{await vi.advanceTimersByTimeAsync(1)});
  expect(hook.result.current.busy).toBe(false);expect(hook.result.current.error?.code).toBe("SERVICE_UNAVAILABLE");expect(hook.result.current.pending).not.toBeNull();expect(requestLearning).toHaveBeenCalledTimes(1);
 } finally {hook.unmount();vi.useRealTimers()}
});
it("a stalled identity proof ends within ten seconds without sending a mutation",async()=>{
 vi.useFakeTimers();vi.mocked(getAuthContext).mockImplementationOnce(()=>new Promise(()=>{}));const hook=renderHook(()=>useLearningCommand(vi.fn()),{wrapper:LearningTestAccount});
 try {act(()=>{void hook.result.current.run({kind:"abandonPractice",id:fixtureID},{})});await act(async()=>{await vi.advanceTimersByTimeAsync(10000)});expect(hook.result.current.busy).toBe(false);expect(hook.result.current.error?.code).toBe("SERVICE_UNAVAILABLE");expect(hook.result.current.pending).toBeNull();expect(requestLearning).not.toHaveBeenCalled()}
 finally {hook.unmount();vi.useRealTimers()}
});
