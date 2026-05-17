import type { FormInstance, InputRef } from "antd";
import type { RefObject } from "react";

/** 从 Form 或 DOM 读取邮箱（兼容浏览器自动填充未写入 Form store 的情况）。 */
export async function resolveEmailFromForm(
  form: FormInstance,
  inputRef?: RefObject<InputRef | null>,
): Promise<string> {
  let email = String(form.getFieldValue("email") ?? "").trim();
  if (!email && inputRef?.current?.input) {
    email = String(inputRef.current.input.value ?? "").trim();
    if (email) {
      form.setFieldsValue({ email });
    }
  }
  if (email) {
    return email;
  }

  try {
    const values = await form.validateFields(["email"]);
    return String(values.email ?? "").trim();
  } catch {
    return "";
  }
}
