-- ============================================================================
-- Translation script: Add English translations for existing experiences
-- ============================================================================
-- This script updates the *_en fields for all experiences
-- Run with: docker exec -i maicivy-postgres psql -U maicivy -d maicivy < translate_experiences_en.sql
-- ============================================================================

BEGIN;

-- Experience 1: Cogesco - Développeur IT Polyvalent
UPDATE experiences
SET
    title_en = 'Versatile IT Developer (C++, VBA, SQL, .Net, Unity3D, AI)',
    description_en = 'Implementation of automation tools with Microsoft Access, rapid and iterative deployment. Creation of a customizable 3D demonstrator for product preview. Comprehensive IT support: AI-automated newsletters, servers and network infrastructure.',
    catchphrase_en = 'Full-stack IT development with automation and 3D visualization',
    functional_description_en = 'Designed and deployed automation solutions to streamline business processes. Built a 3D product visualization system allowing customers to preview customized products in real-time. Managed IT infrastructure and implemented AI-powered communication tools.',
    technical_description_en = 'Developed automation tools using Microsoft Access and VBA for rapid deployment. Created a 3D product configurator using Unity3D and C++. Implemented AI-based newsletter generation system. Administered servers and network infrastructure. Technologies: C++, VBA, SQL, .Net, Unity3D, AI/ML, Microsoft Access.'
WHERE id = '13147d18-2ae4-406e-95d8-b833f202a7b4';

-- Experience 2: Taglabs - Développeur C++ / Unity Mobile
UPDATE experiences
SET
    title_en = 'C++ / Unity Mobile Developer',
    description_en = 'Development of advanced CAD software using point cloud scanning technology (C++, Qt). Software architecture design and user interface creation. Parallel development of a complementary mobile application (Unity, C#). Innovative solutions for complex 3D data processing.',
    catchphrase_en = 'Advanced CAD software and 3D point cloud processing',
    functional_description_en = 'Built professional-grade CAD software for 3D scanning and modeling. Designed intuitive user interfaces for complex 3D data manipulation. Developed a companion mobile app to extend functionality to mobile platforms. Delivered innovative solutions for processing large-scale point cloud data.',
    technical_description_en = 'Developed CAD software in C++ with Qt framework for cross-platform compatibility. Implemented point cloud processing algorithms for 3D scanning technology. Designed scalable software architecture following SOLID principles. Created mobile companion app using Unity and C# for real-time 3D visualization. Technologies: C++, Qt, Unity, C#, 3D graphics, point cloud processing.'
WHERE id = 'ed894759-aac1-4603-90dd-1d06658d1743';

-- Experience 3: Alors Evidemment - Stagiaire Développeur IT
UPDATE experiences
SET
    title_en = 'IT Developer Intern',
    description_en = 'Creation of mobile quiz applications with client-server communication, notably using Unity3D in C#.',
    catchphrase_en = 'Mobile quiz app development with Unity3D',
    functional_description_en = 'Developed interactive mobile quiz applications with real-time client-server communication. Implemented multiplayer features and score tracking systems. Delivered engaging user experiences through gamification.',
    technical_description_en = 'Built mobile quiz applications using Unity3D and C#. Implemented client-server architecture for real-time data synchronization. Developed RESTful APIs for score management and user authentication. Technologies: Unity3D, C#, client-server architecture, mobile development.'
WHERE id = '5a36fcf4-7917-41fe-b440-cf7fac2f5ec0';

-- Experience 4: Cogesco - Stagiaire Développeur VBA
UPDATE experiences
SET
    title_en = 'VBA Developer Intern',
    description_en = 'Creation and deployment of work and employee management software. Creation of automated advertising tools for social networks. Generation of automatic tests based on defined constraints. Microsoft Access in VBA and Automate 8.',
    catchphrase_en = 'Business automation with VBA and Microsoft Access',
    functional_description_en = 'Developed and deployed management software for workforce tracking and employee administration. Created automated social media advertising tools to streamline marketing processes. Implemented constraint-based automatic test generation systems for quality assurance.',
    technical_description_en = 'Built employee management systems using Microsoft Access and VBA. Developed automation tools for social media advertising campaigns. Implemented automatic test generation based on business rules and constraints. Used Automate 8 for workflow automation. Technologies: VBA, Microsoft Access, Automate 8, business process automation.'
WHERE id = '14f87c58-25c8-482c-a18f-5e63286bb2ba';

COMMIT;

-- Verification: Display updated records
SELECT
    company,
    title,
    title_en,
    CASE
        WHEN title_en IS NOT NULL AND title_en != '' THEN '✓'
        ELSE '✗'
    END as "EN Translation"
FROM experiences
ORDER BY start_date DESC;
