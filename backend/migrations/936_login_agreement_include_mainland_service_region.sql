-- Reverse of 927: restore the inclusive Mainland China service-region
-- wording. Idempotent: no-op if the exclusion sentence is already gone.
-- Product decision: LumioAPI serves Mainland China. llms.txt already says so.

UPDATE settings
   SET value = (
         SELECT coalesce(
                  jsonb_agg(
                    CASE
                      WHEN elem->>'id' = 'service-region'
                       AND coalesce(elem->>'content_md', '') LIKE '%不向位于中国大陆地区的用户%'
                      THEN jsonb_set(
                             elem,
                             '{content_md}',
                             to_jsonb($mainland_region$## 服务区域声明 / Service Region

本服务面向全球，包括中国大陆。位于中国大陆地区的用户、通常居住于中国大陆地区的用户，以及中国大陆注册或经营的实体，可以访问、注册、充值并调用 API，具体以服务条款和适用法律为准。

This service is available worldwide, including Mainland China. Users located or ordinarily resident in Mainland China, and entities registered or operating in Mainland China, may access, register, add funds, and use the API, subject to the Terms of Service and applicable law.

如有疑问，请联系 / Questions: admin@lumio.games
$mainland_region$::text)
                           )
                      ELSE elem
                    END
                  ),
                  '[]'::jsonb
                )::text
           FROM jsonb_array_elements(value::jsonb) AS elem
       )
 WHERE key = 'login_agreement_documents'
   AND value LIKE '%不向位于中国大陆地区的用户%';

UPDATE settings
   SET value = '2026-08-24'
 WHERE key = 'login_agreement_updated_at'
   AND EXISTS (
         SELECT 1
           FROM settings s
          WHERE s.key = 'login_agreement_documents'
            AND s.value LIKE '%面向全球，包括中国大陆%'
       );
