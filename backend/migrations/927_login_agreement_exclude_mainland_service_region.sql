-- Reverse of 926: restore the login-agreement "service-region" document
-- when it still includes Mainland China. Idempotent: no-op if the
-- inclusive sentence is gone.

UPDATE settings
   SET value = (
         SELECT coalesce(
                  jsonb_agg(
                    CASE
                      WHEN elem->>'id' = 'service-region'
                       AND (
                             coalesce(elem->>'content_md', '') LIKE '%面向全球，包括中国大陆%'
                          OR coalesce(elem->>'content_md', '') LIKE '%available worldwide, including Mainland China%'
                           )
                      THEN jsonb_set(
                             elem,
                             '{content_md}',
                             to_jsonb($mainland_region$## 服务区域声明 / Service Region

本服务不向位于中国大陆地区的用户、通常居住于中国大陆地区的用户，以及中国大陆注册或经营的实体提供访问、注册或技术支持。

This service is NOT offered to users located or ordinarily resident in Mainland China, or to entities registered or operating in Mainland China.

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
   AND (
         value LIKE '%面向全球，包括中国大陆%'
      OR value LIKE '%available worldwide, including Mainland China%'
       );

UPDATE settings
   SET value = '2026-08-17'
 WHERE key = 'login_agreement_updated_at'
   AND EXISTS (
         SELECT 1
           FROM settings s
          WHERE s.key = 'login_agreement_documents'
            AND s.value LIKE '%不向位于中国大陆地区的用户%'
       );
