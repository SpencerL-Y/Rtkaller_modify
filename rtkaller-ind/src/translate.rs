// Copyright 2021, Developed by Tsinghua Wingtecher Lab and Shumuyulin Ltd, All rights reserved.
use crate::model::HookType;
use crate::prog::{Call, ISRInst, Inst, PtrValue, TaskInst, Value};

const DECL: &str = r"// AUTO-GENERATED by UFUZZ";
const BACKGROUND_TASK: &str = r"int main(void){
    StartOS(OSDEFAULTAPPMODE);
    return 0;
}";

pub fn to_c(inst: &Inst, custom: Option<String>) -> String {
    let mut items = vec![String::from(DECL)];
    if let Some(custom) = custom {
        items.push(custom);
    }

    for (tp, s) in inst.hooks.iter_hook() {
        items.push(trans_hook(tp, s));
    }

    for isr in inst.isr.iter() {
        items.push(trans_isr(isr));
    }

    for t in inst.tasks.iter() {
        items.push(trans_task(t));
    }

    items.push(String::from(BACKGROUND_TASK));

    items.join("\n\n")
}

fn trans_task(t: &TaskInst) -> String {
    format!("TASK({}){{\n{}}}", t.id, trans_seq(&t.seq))
}

fn trans_isr(isr: &ISRInst) -> String {
    let mar = if isr.meta.is_isr1 { "ISR" } else { "ISR2" };

    let c = if let Some(c) = &isr.meta.handler {
        c
    } else {
        &isr.meta.id
    };

    format!("{}({}){{\n{}}}", mar, c, trans_seq(&isr.seq))
}

fn trans_hook(tp: HookType, s: &[Call]) -> String {
    let c = match tp {
        HookType::ERROR => "void ErrorHook(StatusType Error)",
        HookType::STARTUP => "void StartupHook(void)",
        HookType::SHUTDOWN => "void ShutdownHook(StatusType Error)",
        HookType::PRE_TASK => "void PreTaskHook(void)",
        HookType::POST_TASK => "void PostTaskHook(void)",
        _ => unreachable!(),
    };

    format!("{}{{\n{}}}", c, trans_seq(s))
}

fn trans_seq(s: &[Call]) -> String {
    use std::fmt::Write;

    let mut decls = String::new();
    let mut calls = String::new();
    for (i, c) in s.iter().enumerate() {
        let (decl, call) = trans_call(c, i);
        if let Some(decl) = decl {
            writeln!(decls, "{}", decl).unwrap();
        }
        writeln!(calls, "{}", call).unwrap()
    }

    if !decls.is_empty() {
        format!("{}\n{}", decls, calls)
    } else {
        calls
    }
}

fn trans_call(c: &Call, i: usize) -> (Option<String>, String) {
    let mut decls = Vec::new();
    let mut args = Vec::new();

    for (j, arg) in c.args.iter().enumerate() {
        let (decl, a) = trans_value(arg, i, j);
        if let Some(decl) = decl {
            decls.push(decl)
        }
        args.push(a);
    }
    let d = if decls.is_empty() {
        None
    } else {
        Some(decls.join("\n"))
    };

    let args = args.join(", ");
    let call = format!("\t{}({});", c.name, args);

    (d, call)
}

fn trans_value(v: &Value, i: usize, j: usize) -> (Option<String>, String) {
    match v {
        Value::Symbol(s) => (None, s.clone()),
        Value::Num(n) => (None, n.to_string()),
        Value::Ptr(pv) => match pv {
            PtrValue::Out(tp) => {
                let var_name = format!("{}_{}_{}", tp, i, j);
                let decl = format!("\t{} {};", tp, var_name);
                (Some(decl), format!("&{}", var_name))
            }
            PtrValue::Ref(v) => (None, format!("&{}", v)),
            PtrValue::None => (None, "NULL".to_string()),
        },
    }
}
