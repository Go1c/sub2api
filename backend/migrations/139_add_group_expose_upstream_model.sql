-- Add per-group toggle to expose upstream model on user usage records page.
-- expose_upstream_model_to_user: false 时仅管理员能看到上游模型与映射链；
-- true 时用户在该分组对应的使用记录里也能看到 "请求模型 ↳ 上游模型" 展示。
ALTER TABLE groups ADD COLUMN IF NOT EXISTS expose_upstream_model_to_user boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN groups.expose_upstream_model_to_user IS '是否允许用户在使用记录页看到上游模型与映射链；默认 false 仅管理员可见。';
